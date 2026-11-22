package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const gLow = "! M06-2X def2-TZVP def2/J RIJCOSX D3ZERO CPCM(acetonitrile) TightSCF"
const gHigh = "! M06-2X def2-QZVPPD def2/J RIJCOSX D3ZERO CPCM(acetonitrile) TightSCF"
const gResource = "\n%pal\n  nprocs 1\nend\n%maxcore 1000\n"

func gReq() RunRequest {
	return RunRequest{Label: "Glorius synthetic water", Files: []JobFile{
		{Name: "start.xyz", Content: waterXYZ},
		{Name: "goat.inp", Content: "! GOAT XTB2" + gResource + "* xyzfile 0 1 start.xyz\n"},
		{Name: "low.inp", Content: gLow + " Opt Freq" + gResource + "* xyzfile 0 1 goat.globalminimum.xyz\n"},
		{Name: "high.inp", Content: gHigh + gResource + "* xyzfile 0 1 low.xyz\n"},
	}, Stages: []Stage{
		{Name: "GOAT", Input: "goat.inp", Source: "start.xyz", Goat: true},
		{Name: "Low", Input: "low.inp", Source: "goat.globalminimum.xyz", Opt: true, Stationary: "minimum"},
		{Name: "High", Input: "high.inp", Source: "low.xyz", CompositeFrom: "low.inp"},
	}}
}
func TestGloriusPreflightWorkflowAndAllocation(t *testing.T) {
	req := gReq()
	if p := preflight(req); !p.OK {
		t.Fatal(p.Errors)
	}
	if _, _, err := resources(req); err != nil {
		t.Fatal(err)
	}
	if jobs, _, err := planAutoResources([]RunRequest{req}, []bool{true}, 1, resourceSystem{CPU: 1, Supported: true, BudgetMB: 2000}); err != nil || !preflight(jobs[0]).OK {
		t.Fatal(err)
	}
	for _, change := range []func(*RunRequest){
		func(r *RunRequest) { r.Stages[0].Goat = false },
		func(r *RunRequest) { r.Files[1].Content = strings.Split(r.Files[1].Content, "* xyzfile")[0] },
		func(r *RunRequest) {
			r.Files[2].Content = strings.Replace(r.Files[2].Content, "Opt Freq", "Opt Freq IRC", 1)
		},
		func(r *RunRequest) {
			r.Stages[1].Stationary = "ts"
			r.Files[2].Content = strings.Replace(r.Files[2].Content, "Opt Freq", "Opt OptTS Freq", 1)
		},
		func(r *RunRequest) { r.Files[1].Content = strings.Replace(r.Files[1].Content, "XTB2", "XTB1", 1) },
		func(r *RunRequest) {
			r.Files = append(r.Files, JobFile{Name: "goat.globalminimum.xyz", Content: waterXYZ})
		},
		func(r *RunRequest) { r.Files[2].Content = strings.Replace(r.Files[2].Content, "def2/J", "", 1) },
		func(r *RunRequest) { r.Files[3].Content = strings.Replace(r.Files[3].Content, "0 1", "0 3", 1) },
		func(r *RunRequest) { r.Stages[2].CompositeFrom = "goat.inp" },
		func(r *RunRequest) { r.Files[3].Content = strings.Replace(r.Files[3].Content, "TightSCF", "Opt", 1) },
	} {
		r := gReq()
		change(&r)
		if preflight(r).OK {
			t.Fatalf("invalid request accepted: %+v", r)
		}
	}
}
func TestGloriusGOATOutputAndCompositeReport(t *testing.T) {
	req := gReq()
	folder := t.TempDir()
	write := func(n, c string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(folder, n), []byte(c), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range req.Files {
		write(f.Name, f.Content)
	}
	write("goat.out", "Global minimum found!\nWriting structure to goat.globalminimum.xyz\n# Final ensemble info #\nORCA TERMINATED NORMALLY\n")
	write("goat.globalminimum.xyz", waterXYZ)
	if err := validateStage(folder, req.Stages[0]); err != nil {
		t.Fatal(err)
	}
	write("goat.globalminimum.xyz", strings.Replace(waterXYZ, "O 0 0 0", "C 0 0 0", 1))
	if err := validateStage(folder, req.Stages[0]); err == nil {
		t.Fatal("wrong atom composition accepted")
	}
	write("goat.globalminimum.xyz", waterXYZ)
	write("low.xyz", waterXYZ)
	write("low.out", sampleORCA)
	if err := validateStage(folder, req.Stages[1]); err == nil {
		t.Fatal("minimum with imaginary mode accepted")
	}
	normal := strings.Replace(sampleORCA, "-83.22 cm**-1   ***imaginary mode***", "83.22 cm**-1", 1)
	normal = strings.Replace(normal, "Zero point energy", "Temperature ... 298.15 K\nZero point energy", 1)
	write("low.out", normal)
	if err := validateStage(folder, req.Stages[1]); err != nil {
		t.Fatal(err)
	}
	completed := []string{"goat.inp", "low.inp", "high.inp"}
	if rows := gloriusComposites(folder, req, completed); len(rows) != 1 || rows[0].Result != nil {
		t.Fatal("missing SP was invented", rows)
	}
	write("high.out", "SCF CONVERGED AFTER 8 CYCLES\nFINAL SINGLE POINT ENERGY -151.7770000000\nExpectation value of <S**2> : 0.18\nORCA TERMINATED NORMALLY\n")
	if rows := gloriusComposites(folder, req, completed[:2]); rows[0].Result != nil {
		t.Fatal("SP output accepted before its stage passed execution checks")
	}
	rows := gloriusComposites(folder, req, completed)
	expected := -151.777 + (-151.54618270 + 151.55083691)
	if len(rows) != 1 || rows[0].Result == nil || math.Abs(rows[0].Result.CompositeGibbsEh-expected) > 1e-8 {
		t.Fatal(rows)
	}
	resultRow := rows[0].Result
	if resultRow.TemperatureK == nil || *resultRow.TemperatureK != 298.15 ||
		resultRow.CompositeZPECorrectedEh == nil || math.Abs(*resultRow.CompositeZPECorrectedEh-(-151.777+0.02631509)) > 1e-8 ||
		resultRow.CompositeEnthalpyEh == nil || math.Abs(*resultRow.CompositeEnthalpyEh-(-151.777-151.52034633+151.55083691)) > 1e-8 {
		t.Fatalf("missing or incorrect thermochemistry: %+v", resultRow)
	}
	if resultRow.HighS2 == nil || *resultRow.HighS2 != 0.18 ||
		!strings.Contains(strings.Join(resultRow.Warnings, "\n"), "High-level SP OUT에서 spin contamination") {
		t.Fatalf("missing high-level spin diagnostic: %+v", resultRow)
	}
	path, err := writeAnalysisReports(folder, req, completed)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(b), "G_composite") {
		t.Fatal(err, string(b))
	}
	data, err := os.ReadFile(filepath.Join(folder, "results.json"))
	if err != nil {
		t.Fatal(err)
	}
	var result struct{ GloriusComposite []gloriusCompositeRow }
	if err = json.Unmarshal(data, &result); err != nil || len(result.GloriusComposite) != 1 || result.GloriusComposite[0].Result == nil {
		t.Fatal(err, string(data))
	}
	write("high.inp", strings.Replace(req.Files[3].Content, "low.xyz", "start.xyz", 1))
	if gloriusComposites(folder, req, completed)[0].Result != nil {
		t.Fatal("changed SP geometry reference accepted")
	}
	write("high.inp", req.Files[3].Content)
	write("high.out", "FINAL SINGLE POINT ENERGY -151.777\n")
	if gloriusComposites(folder, req, completed)[0].Result != nil {
		t.Fatal("unconverged high SP accepted")
	}
}
func TestGloriusGapRequiresSameStartingSpeciesAndTemperature(t *testing.T) {
	temp := 298.15
	a := SICompositeResult{Atoms: 3, Charge: 0, Multiplicity: 1, CompositeGibbsEh: -100, TemperatureK: &temp}
	b := SICompositeResult{Atoms: 3, Charge: 0, Multiplicity: 3, CompositeGibbsEh: -99.9, TemperatureK: &temp}
	rows := []gloriusCompositeRow{{InitialSource: "moleculeA.xyz", Stationary: "minimum", Result: &a}, {InitialSource: "moleculeB.xyz", Stationary: "minimum", Result: &b}}
	if strings.Contains(gloriusCompositeReport(rows), "Triplet - Singlet composite") {
		t.Fatal("cross-species gap")
	}
	rows[1].InitialSource = rows[0].InitialSource
	if !strings.Contains(gloriusCompositeReport(rows), "Triplet - Singlet composite") {
		t.Fatal("same-source gap missing")
	}
	differentTemperature := 300.0
	for _, temperature := range []*float64{nil, &differentTemperature} {
		b.TemperatureK = temperature
		report := gloriusCompositeReport(rows)
		if strings.Contains(report, "Triplet - Singlet composite") || !strings.Contains(report, "동일한 Temperature") {
			t.Fatal(report)
		}
	}
}

func TestGloriusQueueActualMockChildPipeline(t *testing.T) {
	q := mockQueue(t)
	t.Setenv("ORCA_NOVA_GLORIUS_MOCK", "1")
	q.Limits = QueueLimits{Concurrency: 1, CPU: 1, RAMMB: 2000}
	good := addEntry(t, q, gReq())
	badReq := gReq()
	badReq.Label = "missing GOAT geometry"
	badReq.Files[1].Content += "\n# MOCK_NO_GOAT_XYZ\n"
	bad := addEntry(t, q, badReq)
	if err := q.action("", "resume"); err != nil {
		t.Fatal(err)
	}
	waitQueue(t, q, func(es []*QueueEntry) bool { return es[0].State == "Completed" && es[1].State == "Failed" })
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(good.Job.CompletedStages) != 3 || len(bad.Job.CompletedStages) != 0 {
		t.Fatal("checkpoint mismatch")
	}
	for _, n := range []string{"goat.globalminimum.xyz", "low.xyz", "high.out", "results.json", "analysis.txt"} {
		if _, err := os.Stat(filepath.Join(good.Job.Folder, n)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(bad.Job.Folder, "low.out")); !os.IsNotExist(err) {
		t.Fatal("downstream DFT launched with missing GOAT XYZ", err)
	}
	txt, err := os.ReadFile(filepath.Join(good.Job.Folder, "analysis.txt"))
	if err != nil || !strings.Contains(string(txt), "G_composite") {
		t.Fatal(err, string(txt))
	}
}
