package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileRejectsSolventBlockMismatchAndTDDFT(t *testing.T) {
	s := profileAnalysis("s.out", "PBE0 def2-SVP Opt", 1, -100)
	tr := profileAnalysis("t.out", "PBE0 def2-SVP Opt UKS", 3, -99.9)
	s.Settings = "! pbe0 def2-svp opt\n%cpcm\n epsilon 36\nend"
	tr.Settings = "! pbe0 def2-svp opt uks\n%cpcm\n epsilon 80\nend"
	req := ScientificProfileRequest{Kind: "adiabatic", Energy: "Electronic", Confirmed: true, Sources: map[string]string{"S0": "s", "T1": "t"}}
	if _, err := scientificProfile(req, map[string]OrcaAnalysis{"s": s, "t": tr}); err == nil {
		t.Fatal("solvent block mismatch accepted")
	}
	tr.Settings = s.Settings + "\n%tddft\n iroot 1\nend"
	if _, err := scientificProfile(req, map[string]OrcaAnalysis{"s": s, "t": tr}); err == nil {
		t.Fatal("excited-state root accepted as UKS triplet")
	}
	tr.Settings = s.Settings
	if _, err := scientificProfile(req, map[string]OrcaAnalysis{"s": s, "t": tr}); err != nil {
		t.Fatal(err)
	}
}
func TestProfileMissingCompositionFailsClosed(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	writeProfileFixture(t, a, 1, -100)
	writeProfileFixture(t, b, 3, -99.9)
	if err := os.Remove(filepath.Join(b, "molecule.xyz")); err != nil {
		t.Fatal(err)
	}
	app := &App{queue: &JobQueue{Jobs: []*QueueEntry{{ID: "a", Job: &Job{Folder: a}}, {ID: "b", Job: &Job{Folder: b}}}}}
	req := ScientificProfileRequest{Kind: "adiabatic", Energy: "Electronic", Confirmed: true, Sources: map[string]string{"S0": "same.out", "T1": "same.out"}, Jobs: map[string]string{"S0": "a", "T1": "b"}}
	if _, _, err := app.resolveProfileSources(req); err == nil {
		t.Fatal("missing real composition accepted")
	}
}

func TestProfileMinimumAndSpinWarnings(t *testing.T) {
	s := profileAnalysis("s.out", "PBE0 def2-SVP Opt", 1, -100)
	tr := profileAnalysis("t.out", "PBE0 def2-SVP Opt UKS", 3, -99.9)
	one := 1
	s.ImaginaryCount = &one
	req := ScientificProfileRequest{Kind: "adiabatic", Energy: "Electronic", Confirmed: true, Sources: map[string]string{"S0": "s", "T1": "t"}}
	if _, err := scientificProfile(req, map[string]OrcaAnalysis{"s": s, "t": tr}); err == nil {
		t.Fatal("saddle point accepted as minimum")
	}
	zero := 0
	s.ImaginaryCount = &zero
	s2 := 3.2
	tr.S2 = &s2
	got, err := scientificProfile(req, map[string]OrcaAnalysis{"s": s, "t": tr})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range got.Warnings {
		if strings.Contains(w, "Spin Contamination") {
			found = true
		}
	}
	if !found {
		t.Fatal("spin contamination warning missing")
	}
}
