package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func mockHATStage(inp string) int {
	contents, err := os.ReadFile(inp)
	if err != nil {
		return 2
	}
	text := string(contents)
	h := xyzHeader.FindStringSubmatch(text)
	if len(h) < 4 {
		return 3
	}
	source, err := os.ReadFile(h[3])
	if err != nil {
		return 4
	}
	keywords := keywordLine(text)
	if !keywords["m062x"] || keywords["m06-2x"] {
		return 5
	}
	stem := strings.TrimSuffix(inp, ".inp")
	if err := os.WriteFile(stem+".launched", nil, 0600); err != nil {
		return 6
	}
	fmt.Println("Program Version 6.0.1")
	fmt.Println("SCF CONVERGED AFTER 10 CYCLES")
	fmt.Println("FINAL SINGLE POINT ENERGY -75.1234")
	if keywords["opt"] || keywords["neb-ts"] {
		suffix := ".xyz"
		frequency := 123.45
		if keywords["neb-ts"] {
			suffix = "_NEB-TS_converged.xyz"
			frequency = -123.45
			// An intermediate image must never satisfy the IRC dependency.
			if err := os.WriteFile(stem+"_NEB-CI_converged.xyz", source, 0600); err != nil {
				return 7
			}
			if strings.Contains(text, "MOCK_WRONG_TS_FILENAME") {
				suffix = ".xyz"
			}
			if strings.Contains(text, "MOCK_INVALID_TS_MODE") {
				frequency = 123.45
			}
		}
		if err := os.WriteFile(stem+suffix, source, 0600); err != nil {
			return 8
		}
		fmt.Println("THE OPTIMIZATION HAS CONVERGED")
		fmt.Printf("VIBRATIONAL FREQUENCIES\n  0: %.2f cm**-1\n  1: 999.99 cm**-1\nNORMAL MODES\n", frequency)
		fmt.Println("Temperature ... 298.15 K\nFinal Gibbs free energy ... -75.10000000 Eh")
	}
	if keywords["irc"] {
		for i, name := range []string{"r.xyz", "p.xyz"} {
			raw, err := os.ReadFile(name)
			if err != nil {
				return 9
			}
			if err = os.WriteFile(stem+[]string{"_IRC_F.xyz", "_IRC_B.xyz"}[i], raw, 0600); err != nil {
				return 10
			}
		}
	}
	fmt.Fprintln(os.Stderr, "mock stderr")
	fmt.Println("ORCA TERMINATED NORMALLY")
	return 0
}

func unpackHATTestBundle(t *testing.T, req RunRequest, runner io.Reader) string {
	t.Helper()
	var buf bytes.Buffer
	if err := writeHATZip(&buf, req, runner); err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "한글 space & (test) ! %")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, f := range z.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, f.Name), raw, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range hatLaunchers {
		raw, err := os.ReadFile(filepath.Join(dir, file.filename))
		if err != nil || !strings.Contains(string(raw), "--hat-run "+file.mode+" >") {
			t.Fatalf("%s: %v", file.filename, err)
		}
	}
	return dir
}

func latestHATRun(t *testing.T, dir string, result io.Reader) string {
	t.Helper()
	raw, err := io.ReadAll(result)
	if err != nil {
		t.Fatal(err)
	}
	message := string(raw)
	for _, line := range strings.Split(message, "\n") {
		if folder, ok := strings.CutPrefix(line, "Run folder: "); ok {
			return strings.TrimSpace(folder)
		}
	}
	t.Fatalf("run folder not reported under %s: %s", dir, message)
	return ""
}

func TestHATBundleRunsAndReusesValidatedEndpoints(t *testing.T) {
	t.Setenv("ORCA_NOVA_HAT_MOCK", "1")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := unpackHATTestBundle(t, hatTestRequest(), strings.NewReader("not used by Linux"))
	ctx := context.Background()
	var out bytes.Buffer
	run := func(mode, previous string) (string, error) {
		out.Reset()
		err := runHATBundle(ctx, mode, dir, exe, previous, &out)
		return latestHATRun(t, dir, &out), err
	}
	inp := filepath.Join(dir, "ts.inp")
	good, err := os.ReadFile(inp)
	if err != nil {
		t.Fatal(err)
	}
	for _, failure := range []string{"MOCK_WRONG_TS_FILENAME", "MOCK_INVALID_TS_MODE"} {
		if err = os.WriteFile(inp, append(append([]byte(nil), good...), []byte("\n# "+failure+"\n")...), 0600); err != nil {
			t.Fatal(err)
		}
		folder, err := run("all", "")
		if err == nil {
			t.Fatal("Step 3 failure accepted")
		}
		if _, e := os.Stat(filepath.Join(folder, "irc.launched")); !os.IsNotExist(e) {
			t.Fatal("IRC ran after invalid TS", e)
		}
		if _, e := os.Stat(filepath.Join(folder, "ts.err")); e != nil {
			t.Fatal("stderr not retained", e)
		}
		if failure == "MOCK_WRONG_TS_FILENAME" {
			if err = os.WriteFile(inp, good, 0600); err != nil {
				t.Fatal(err)
			}
			next, err := run("step3", folder)
			if err != nil {
				t.Fatal(err)
			}
			if next == folder {
				t.Fatal("previous output overwritten")
			}
			restarted, err := run("resume", folder)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = os.Stat(filepath.Join(restarted, "r.launched")); !os.IsNotExist(err) {
				t.Fatal("Resume reran the verified reactant")
			}

			for _, name := range []string{"r.launched", "p.launched"} {
				if _, err := os.Stat(filepath.Join(next, name)); !os.IsNotExist(err) {
					t.Fatalf("%s was rerun", name)
				}
			}
			jsonReport, err := os.ReadFile(filepath.Join(next, "results.json"))
			if err != nil || !strings.Contains(string(jsonReport), `"ForwardKcalMol": 0`) {
				t.Fatalf("valid zero barrier missing: %s %v", jsonReport, err)
			}
			report, err := os.ReadFile(filepath.Join(next, "analysis.txt"))
			if err != nil || !strings.Contains(string(report), "Forward → Reactant") {
				t.Fatalf("IRC endpoint not reported: %v", err)
			}
			irc, err := run("irc", next)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(irc, "ts.launched")); !os.IsNotExist(err) {
				t.Fatal("TS rerun in IRC Only")
			}
			if err = os.WriteFile(filepath.Join(next, "r.out"), nil, 0600); err != nil {
				t.Fatal(err)
			}
			if err = runHATBundle(ctx, "step3", dir, exe, next, io.Discard); err == nil {
				t.Fatal("changed checkpoint accepted")
			}
		}
	}
}

func TestHATWindowsBATRunsPortableGUIExecutable(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows BAT required")
	}
	t.Setenv("ORCA_NOVA_HAT_MOCK", "1")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("ORCA_EXE", exe)
	runnerPath := filepath.Join(t.TempDir(), "runner.exe")
	cmd := exec.Command("go", "build", "-ldflags", "-H windowsgui", "-o", runnerPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, out)
	}
	runner, err := os.Open(runnerPath)
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Close()
	dir := unpackHATTestBundle(t, hatTestRequest(), runner)
	cmd = exec.Command("cmd.exe", "/d", "/c", "RUN_ALL_1_TO_4.bat")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("BAT run: %v: %s", err, out)
	}
	dirs, err := os.ReadDir(filepath.Join(dir, "HAT_results"))
	if err != nil || len(dirs) != 1 {
		t.Fatalf("BAT did not create one run: %v: %s", err, out)
	}
	if _, err := os.Stat(filepath.Join(dir, "HAT_results", dirs[0].Name(), "irc.out")); err != nil {
		t.Fatal(err)
	}
}
