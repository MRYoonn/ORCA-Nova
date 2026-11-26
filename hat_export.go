package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const hatPlanName = "hat-plan.json"

var hatLaunchers = []struct{ filename, mode string }{
	{"RUN_ALL_1_TO_4.bat", "all"},
	{"RUN_STEP3_TO_4.bat", "step3"},
	{"RUN_IRC_ONLY.bat", "irc"},
	{"RESUME_FAILED_STAGE.bat", "resume"},
}

const hatReadme = `ORCA Nova D9 — standalone workflow

Extract the entire ZIP. Set ORCA_EXE to the absolute path of orca.exe (PATH also works).
Set ORCA_RESUME_DIR to an absolute previous run folder for Step 3–4, IRC Only or Resume.
Examples in PowerShell (keep quotes for paths with spaces or Korean characters):
  $env:ORCA_EXE='C:\ORCA\orca.exe'
  $env:ORCA_RESUME_DIR='C:\path\to\this\ZIP\HAT_results\HAT_run_12345'
  .\RUN_STEP3_TO_4.bat

Each run writes a new HAT_results/HAT_run_* folder. Read HAT_RUN.log and
ORCA_Nova_Analysis.txt. Match atom indices and inspect normal modes, IRC
endpoints and spin before interpreting barriers. Earlier results are reused
only if their checked geometry, INP, OUT and atom map still agree. Keep
the verified JSON files with the run. Failed source files are preserved.
The runner needs an ORCA 6 installation and its MPI runtime; Nova installation
and WebView2 are unnecessary. Edit the INPs in the extracted folder before
running; the runner rechecks all stages before launching ORCA.
`

func writeHATZip(dst io.Writer, req RunRequest, runner io.Reader) error {
	z := zip.NewWriter(dst)
	add := func(name string, r io.Reader) error {
		w, err := z.Create(name)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, r)
		return err
	}
	for _, f := range req.Files {
		if err := add(f.Name, strings.NewReader(f.Content)); err != nil {
			return err
		}
	}
	files := append([]JobFile(nil), req.Files...)
	for i := range files {
		files[i].Content = ""
	}
	req.Files = files
	req.AutoCPU = 0
	req.HATRestart = nil
	plan, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return err
	}
	if err := add(hatPlanName, strings.NewReader(string(plan))); err != nil {
		return err
	}
	for _, launcher := range hatLaunchers {
		bat := "@echo off\r\nsetlocal DisableDelayedExpansion\r\npushd \"%~dp0\" || exit /b 1\r\n" +
			"start \"\" /b /wait \".\\HAT_RUNNER.exe\" --hat-run " + launcher.mode + " > HAT_RUN.log 2>&1\r\n" +
			"set \"HAT_EXIT=%errorlevel%\"\r\ntype HAT_RUN.log\r\npopd\r\nexit /b %HAT_EXIT%\r\n"
		if err := add(launcher.filename, strings.NewReader(bat)); err != nil {
			return err
		}
	}
	if err := add("README_HAT.txt", strings.NewReader(hatReadme)); err != nil {
		return err
	}
	if err := add("HAT_RUNNER.exe", runner); err != nil {
		return err
	}
	return z.Close()
}

func (a *App) exportHAT(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 405, "POST required")
		return
	}
	if runtime.GOOS != "windows" {
		fail(w, 501, "Windows Portable 앱에서 HAT 실행 ZIP을 내보내세요")
		return
	}
	var req RunRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 12<<20)).Decode(&req); err != nil {
		fail(w, 400, "HAT export 입력을 확인하세요")
		return
	}
	req = normalizeORCARequest(req)
	req.HATRestart = nil
	req.AutoCPU = 0
	if req.HATMapping == nil {
		fail(w, 400, "HAT 원자 대응을 확인하세요")
		return
	}
	if check := preflight(req); !check.OK {
		fail(w, 422, strings.Join(check.Errors, " / "))
		return
	}
	path, err := os.Executable()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	runner, err := os.Open(path)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer runner.Close()
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="D9_HAT_Workflow.zip"`)
	if err := writeHATZip(w, req, runner); err != nil {
		fmt.Fprintln(os.Stderr, "HAT export:", err)
	}
}

func runHATBundle(ctx context.Context, mode, dir, orcaExe, previous string, output io.Writer) error {
	raw, err := readGloriusFile(dir, hatPlanName)
	if err != nil {
		return err
	}
	var req RunRequest
	if err = json.Unmarshal([]byte(raw), &req); err != nil {
		return err
	}
	if req.HATMapping == nil || len(req.Files) > 80 {
		return fmt.Errorf("invalid HAT manifest")
	}
	for i, f := range req.Files {
		text, err := readGloriusFile(dir, f.Name)
		if err != nil {
			return err
		}
		req.Files[i].Content = text
	}
	req = normalizeORCARequest(req)
	req.HATRestart = nil
	if mode != "all" {
		req.HATRestart = &hatRestart{Mode: mode, Folder: previous}
	}
	if report := preflight(req); !report.OK {
		return fmt.Errorf("preflight: %s", strings.Join(report.Errors, " / "))
	}
	orcaExe = strings.Trim(orcaExe, `" `)
	if orcaExe == "" {
		orcaExe = "orca.exe"
	}
	if !filepath.IsAbs(orcaExe) {
		orcaExe, err = exec.LookPath(orcaExe)
		if err != nil {
			return err
		}
	}
	if info, err := os.Stat(orcaExe); err != nil || info.IsDir() {
		return fmt.Errorf("ORCA_EXE: %s", orcaExe)
	}
	root := filepath.Join(dir, "HAT_results")
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	from, count, err := validateHATRestart(root, req)
	if err != nil {
		return err
	}
	folder, err := os.MkdirTemp(root, "HAT_run_")
	if err != nil {
		return err
	}
	fmt.Fprintln(output, "Run folder:", folder)
	for _, f := range req.Files {
		if err := os.WriteFile(filepath.Join(folder, f.Name), []byte(f.Content), 0600); err != nil {
			return err
		}
	}
	if err := writeHATMapping(folder, req); err != nil {
		return err
	}
	completed, err := copyHATCheckpoints(from, folder, req, count)
	if err != nil {
		return err
	}
	job := &Job{Folder: folder, State: "running", Started: time.Now().Format(time.RFC3339), StageCount: len(req.Stages), CompletedStages: completed}
	app := &App{}
	app.runJob(ctx, orcaExe, req, job)
	if job.State != "completed" || job.ReportFile == "" {
		return fmt.Errorf("%s (see %s)", job.Message, folder)
	}
	fmt.Fprintln(output, job.ReportFile)
	return nil
}

func runHATCommand(mode string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	return runHATBundle(ctx, mode, dir, os.Getenv("ORCA_EXE"), os.Getenv("ORCA_RESUME_DIR"), os.Stdout)
}
