package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleORCA = `------------------------------
SCF CONVERGED AFTER 8 CYCLES
FINAL SINGLE POINT ENERGY     -151.5508369100
THE OPTIMIZATION HAS CONVERGED
-----------------------
MULLIKEN ATOMIC CHARGES AND SPIN POPULATIONS
-----------------------
  0 C :  -0.202000  0.420000
  1 S :   0.233000  0.580000
Sum of atomic charges:  0.000000
-----------------------
LOEWDIN ATOMIC CHARGES AND SPIN POPULATIONS
-----------------------
  0 C : -0.110000  0.600000
  1 S :  0.110000  0.400000
-----------------------
VIBRATIONAL FREQUENCIES
-----------------------
  0:       0.00 cm**-1
  1:      -83.22 cm**-1   ***imaginary mode***
  2:      435.79 cm**-1
NORMAL MODES
Summary of contributions to the inner energy U:
Zero point energy                ...      0.02631509 Eh      16.51 kcal/mol
Total Enthalpy                    ...   -151.52034633 Eh
Final entropy term                ...      0.02583637 Eh
Final Gibbs free energy         ...   -151.54618270 Eh
----------------------
TD-DFT EXCITED STATES (SINGLETS)
----------------------
STATE  1:  E=   0.085974 au      2.339 eV    18866.4 cm**-1
TD-DFT EXCITED STATES (TRIPLETS)
STATE  1:  E=   0.071343 au      1.941 eV    15653.0 cm**-1
ORCA TERMINATED NORMALLY
`

func TestParseORCAThermoSpinExcitedAndImaginary(t *testing.T) {
	inp := "! PBE0 D4 def2-SVP Opt Freq\n* xyzfile 0 3 molecule.xyz\n"
	a, e := ParseOrcaOutput(strings.NewReader(sampleORCA), "mol.out", inp)
	if e != nil {
		t.Fatal(e)
	}
	if !a.Normal || !a.Optimized || !a.SCFConverged {
		t.Fatalf("status mismatch: %+v", a)
	}
	if a.Multiplicity != 3 || a.Electronic == nil || *a.Electronic != -151.55083691 {
		t.Fatalf("wrong electronic: %+v", a)
	}
	if a.Enthalpy == nil || *a.Enthalpy != -151.52034633 {
		t.Fatalf("wrong H: %+v", a.Enthalpy)
	}
	if a.Gibbs == nil || *a.Gibbs != -151.54618270 {
		t.Fatalf("wrong G: %+v", a.Gibbs)
	}
	if len(a.Roots) != 2 || a.Roots[0].Kind != "S" || a.Roots[1].Kind != "T" {
		t.Fatalf("wrong roots: %+v", a.Roots)
	}
	if len(a.Populations["Mulliken"]) != 2 || *a.Populations["Mulliken"][0].Spin != 0.42 {
		t.Fatal(a.Populations)
	}
	if len(a.Frequencies) != 3 || a.Frequencies[1] != -83.22 {
		t.Fatal(a.Frequencies)
	}
	txt := formatAnalysis(a, Stage{Name: "T1 Opt", Input: "mol.inp", Source: "mol.xyz"})
	for _, key := range []string{"Enthalpy", "Gibbs Free Energy", "Spin Population", "-83.22", "S1", "T1", "원자 번호는 0부터"} {
		if !strings.Contains(txt, key) {
			t.Fatalf("missing %q", key)
		}
	}
}
func TestNoInventedDataAndPartialOutput(t *testing.T) {
	a, e := ParseOrcaOutput(strings.NewReader("FINAL SINGLE POINT ENERGY -10.000000\n"), "partial.out", "* xyzfile 0 1 x.xyz")
	if e != nil {
		t.Fatal(e)
	}
	if a.Normal || a.Gibbs != nil || a.Enthalpy != nil || len(a.Roots) != 0 {
		t.Fatal("invented result")
	}
	txt := formatAnalysis(a, Stage{Name: "SP"})
	if !strings.Contains(txt, "계산되지 않음") || !strings.Contains(txt, "정상 종료 문구가 없습니다") {
		t.Fatal(txt)
	}
}
func TestWriteWorkflowReportNoMismatchedMethod(t *testing.T) {
	dir := t.TempDir()
	input0 := "! PBE0 def2-SVP Opt Freq\n* xyzfile 0 1 mol.xyz\n"
	inputT := "! PBE0 def2-SVP Opt Freq\n* xyzfile 0 3 mol.xyz\n"
	for name, contents := range map[string]string{"s0.inp": input0, "t1.inp": inputT, "s0.out": "FINAL SINGLE POINT ENERGY -100.000000\nTHE OPTIMIZATION HAS CONVERGED\nORCA TERMINATED NORMALLY\n", "t1.out": "FINAL SINGLE POINT ENERGY -99.900000\nTHE OPTIMIZATION HAS CONVERGED\nORCA TERMINATED NORMALLY\n"} {
		if e := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0600); e != nil {
			t.Fatal(e)
		}
	}
	req := RunRequest{Label: "test workflow", Stages: []Stage{{Name: "S0 Opt", Input: "s0.inp", Source: "mol.xyz", Opt: true}, {Name: "T1 Opt", Input: "t1.inp", Source: "mol.xyz", Opt: true}}}
	path, e := writeAnalysisReports(dir, req)
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	txt := string(b)
	if !strings.Contains(txt, "+2.721") || !strings.Contains(txt, "Electronic Energy 차이") || !strings.Contains(txt, "Gibbs Free Energy 차이가 아닙니다") {
		t.Fatal(txt)
	}
	if _, e := os.Stat(filepath.Join(dir, "s0_analysis.txt")); e != nil {
		t.Fatal(e)
	}
}
func TestNoAdiabaticFromOnlyVertical(t *testing.T) {
	dir := t.TempDir()
	for name, contents := range map[string]string{"vert.inp": "! PBE0 TDDFT\n* xyzfile 0 1 mol.xyz\n", "vert.out": "FINAL SINGLE POINT ENERGY -250.1234\nTD-DFT EXCITED STATES (TRIPLETS)\nSTATE 1: E= 0.07 au 1.9048 eV\nORCA TERMINATED NORMALLY\n"} {
		if e := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0600); e != nil {
			t.Fatal(e)
		}
	}
	path, e := writeAnalysisReports(dir, RunRequest{Stages: []Stage{{Name: "Vertical", Input: "vert.inp"}}})
	if e != nil {
		t.Fatal(e)
	}
	bytes, _ := os.ReadFile(path)
	if strings.Contains(string(bytes), "최적화 S₀(Singlet)/Triplet 상태 Electronic Energy 차이:") {
		t.Fatal("invalid adiabatic comparison")
	}
}
