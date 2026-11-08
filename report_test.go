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
	for name, contents := range map[string]string{"s0.inp": input0, "t1.inp": inputT, "s0.out": "SCF CONVERGED AFTER 4 CYCLES\nFINAL SINGLE POINT ENERGY -100.000000\nTHE OPTIMIZATION HAS CONVERGED\nORCA TERMINATED NORMALLY\n", "t1.out": "SCF CONVERGED AFTER 4 CYCLES\nFINAL SINGLE POINT ENERGY -99.900000\nTHE OPTIMIZATION HAS CONVERGED\nORCA TERMINATED NORMALLY\n"} {
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

func TestExtendedThermochemistryAndReportArtifacts(t *testing.T) {
	input := "! PBE0 def2-SVP Freq\n* xyzfile 0 1 water.xyz\n"
	out := `SCF CONVERGED AFTER 4 CYCLES
FINAL SINGLE POINT ENERGY -76.400000
Total Enthalpy ... -76.350000 Eh
Final Gibbs free energy ... -76.370000 Eh
Thermal correction to Energy ... 0.049000 Eh
Total entropy ... 0.000067081 Eh/K
Temperature : 298.15 K
ORBITAL ENERGIES
 0 2.0000 -0.50 -13.6057
 1 2.0000 -0.25 -6.8028
 2 0.0000 0.05 1.3606
ORCA TERMINATED NORMALLY
`
	a, err := ParseOrcaOutput(strings.NewReader(out), "water.out", input)
	if err != nil {
		t.Fatal(err)
	}
	if a.ThermalCorrection == nil || *a.ThermalCorrection != 0.049 || a.Entropy == nil || *a.Entropy != 0.000067081 || a.ThermalEnthalpyCorrection == nil {
		t.Fatalf("missing thermo: %+v", a)
	}
	if a.Geometry != "water.xyz" || a.Status != "Completed" || !a.Frontier.FoundH || a.Frontier.HOMO != -6.8028 || a.Frontier.LUMO != 1.3606 {
		t.Fatalf("metadata/frontier: %+v", a)
	}
	dir := t.TempDir()
	for name, contents := range map[string]string{"water.inp": input, "water.out": out} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, err = writeAnalysisReports(dir, RunRequest{Label: "water", Stages: []Stage{{Name: "Freq", Input: "water.inp"}, {Name: "Not started", Input: "next.inp"}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"analysis.txt", "results.json", "water_analysis.txt", "ORCA_Nova_Analysis.txt"} {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || len(b) == 0 {
			t.Fatalf("%s: %s", name, err)
		}
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "results.json"))
	if !strings.Contains(string(raw), `"Not calculated"`) || !strings.Contains(string(raw), `"ThermalCorrection": 0.049`) {
		t.Fatal(string(raw))
	}
}

func TestRootWarningsAndConservativeOpenShellFrontier(t *testing.T) {
	out := `SCF CONVERGED AFTER 8 CYCLES
FINAL SINGLE POINT ENERGY -100.00
Expectation value of <S**2> : 2.60
WARNING ROOT FOLLOWING FAILED
VIBRATIONAL FREQUENCIES
 0: -200.00 cm**-1
NORMAL MODES
ORBITAL ENERGIES
 0 2.0000 -0.50 -13.6057
 1 1.0000 -0.25 -6.8028
 2 0.0000 0.05 1.3606
ORCA TERMINATED NORMALLY
`
	a, err := ParseOrcaOutput(strings.NewReader(out), "trip.out", "! PBE0 def2-SVP Freq\n* xyzfile 0 3 x.xyz\n")
	if err != nil {
		t.Fatal(err)
	}
	if a.Frontier.FoundH || a.ImaginaryCount == nil || *a.ImaginaryCount != 1 {
		t.Fatalf("frontier/frequency: %+v", a)
	}
	text := strings.Join(a.Warnings, "\n")
	for _, key := range []string{"Root Following", "Spin Contamination", "Minimum"} {
		if !strings.Contains(text, key) {
			t.Fatalf("missing %s: %s", key, text)
		}
	}
	ts, err := ParseOrcaOutput(strings.NewReader(out), "ts.out", "! PBE0 OptTS Freq\n* xyzfile 0 3 x.xyz\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(ts.Warnings, "\n"), "Frequency 검토") {
		t.Fatal("one imaginary should not trigger count mismatch for OptTS")
	}
}

func TestSingletAbsorptionDataNeverAssignedToTriplet(t *testing.T) {
	out := `SCF CONVERGED AFTER 4 CYCLES
FINAL SINGLE POINT ENERGY -100.0
TD-DFT EXCITED STATES (SINGLETS)
STATE 1: E= 0.100000 au 2.7211 eV
ABSORPTION SPECTRUM VIA TRANSITION ELECTRIC DIPOLE MOMENTS
State Energy Wavelength fosc T2 TX TY TZ
1 21947.46 455.63 0.010000 0.0 0.0 0.0 0.0
TD-DFT EXCITED STATES (TRIPLETS)
STATE 1: E= 0.070000 au 1.9048 eV
ORCA TERMINATED NORMALLY
`
	a, err := ParseOrcaOutput(strings.NewReader(out), "td.out", "! PBE0 TDDFT\n* xyzfile 0 1 x.xyz\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Roots) != 2 || a.Roots[0].OscillatorStrength == nil || *a.Roots[0].OscillatorStrength != 0.01 || a.Roots[0].WavelengthNM == nil || *a.Roots[0].WavelengthNM != 455.63 || a.Roots[1].OscillatorStrength != nil {
		t.Fatalf("incorrect absorption mapping: %+v", a.Roots)
	}
}

func TestReportDoesNotCompareDifferentSolventBlocks(t *testing.T) {
	dir := t.TempDir()
	for _, state := range []string{"s0", "t1"} {
		mult, solvent := "1", "water"
		if state == "t1" {
			mult, solvent = "3", "THF"
		}
		inp := "! PBE0 def2-SVP Opt\n%cpcm\n solvent \"" + solvent + "\"\nend\n* xyzfile 0 " + mult + " x.xyz\n"
		out := "FINAL SINGLE POINT ENERGY -100.0\nSCF CONVERGED AFTER 4 CYCLES\nTHE OPTIMIZATION HAS CONVERGED\nORCA TERMINATED NORMALLY\n"
		os.WriteFile(filepath.Join(dir, state+".inp"), []byte(inp), 0600)
		os.WriteFile(filepath.Join(dir, state+".out"), []byte(out), 0600)
	}
	path, err := writeAnalysisReports(dir, RunRequest{Stages: []Stage{{Input: "s0.inp"}, {Input: "t1.inp"}}})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if strings.Contains(string(b), "최적화 S₀(Singlet)/Triplet 상태 Electronic Energy 차이:") || !strings.Contains(string(b), "자동 계산 제외") {
		t.Fatal(string(b))
	}
}

func TestModernAbsorptionAndHirshfeldPopulation(t *testing.T) {
	out := `SCF CONVERGED AFTER 3 CYCLES
FINAL SINGLE POINT ENERGY -20.0
HIRSHFELD ANALYSIS
ATOM CHARGE SPIN
0 C -0.150000 0.250000
1 H 0.150000 0.750000
TD-DFT EXCITED STATES (SINGLETS)
STATE 1: E= 0.100000 au 2.7211 eV
ABSORPTION SPECTRUM VIA TRANSITION ELECTRIC DIPOLE MOMENTS
Transition Energy Energy Wavelength fosc(D2) D2 DX DY DZ
0-1A -> 1-1A 2.7211 21947.46 455.63 0.012345 0.0 0.0 0.0 0.0
ORCA TERMINATED NORMALLY
`
	a, err := ParseOrcaOutput(strings.NewReader(out), "td.out", "! PBE0 TDDFT\n* xyzfile 0 1 x.xyz\n")
	if err != nil {
		t.Fatal(err)
	}
	rows := a.Populations["Hirshfeld"]
	if len(rows) != 2 || rows[0].Spin == nil || *rows[0].Spin != 0.25 || a.Roots[0].OscillatorStrength == nil || *a.Roots[0].OscillatorStrength != 0.012345 {
		t.Fatalf("modern table failed: %+v", a)
	}
}

func TestErrorTerminationInvalidatesEarlierNormalMarker(t *testing.T) {
	out := "SCF CONVERGED AFTER 4 CYCLES\nFINAL SINGLE POINT ENERGY -100.0\nORCA TERMINATED NORMALLY\nORCA finished by error termination in SCF\n"
	a, err := ParseOrcaOutput(strings.NewReader(out), "partial.out", "! PBE0 def2-SVP\n* xyzfile 0 1 x.xyz\n")
	if err != nil {
		t.Fatal(err)
	}
	if a.Normal || a.Status != "Failed" || a.Electronic == nil {
		t.Fatalf("partial result must remain failed and retain reported measurements: %+v", a)
	}
}
