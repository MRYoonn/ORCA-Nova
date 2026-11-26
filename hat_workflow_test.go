package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func hatTestRequest() RunRequest {
	a := "3\nreactant\nC 0 0 0\nC 1.5 0 0\nH 0 1 0\n"
	b := "3\nproduct\nC 0 0 0\nC 1.5 0 0\nH 1.5 1 0\n"
	base := "! UKS M062X def2-TZVP def2/J RIJCOSX TightSCF CPCM(dmf) "
	freq := "\n%freq\n Temp 298.15\nend\n"
	return RunRequest{HATMapping: &hatMapping{H: 2, Donor: 0, Acceptor: 1}, Files: []JobFile{
		{Name: "reactant.xyz", Content: a}, {Name: "product.xyz", Content: b},
		{Name: "r.inp", Content: base + "Opt Freq" + freq + "* xyzfile 0 2 reactant.xyz\n"},
		{Name: "p.inp", Content: base + "Opt Freq" + freq + "* xyzfile 0 2 product.xyz\n"},
		{Name: "ts.inp", Content: base + "Freq NEB-TS" + freq + "%neb\n NEB_END_XYZFILE \"p.xyz\"\nend\n* xyzfile 0 2 r.xyz\n"},
		{Name: "irc.inp", Content: base + "IRC\n%irc\n Direction both\nend\n* xyzfile 0 2 ts_NEB-TS_converged.xyz\n"},
	}, Stages: []Stage{
		{Name: "R", Input: "r.inp", Source: "reactant.xyz", Opt: true, HAT: "reactant"},
		{Name: "P", Input: "p.inp", Source: "product.xyz", Opt: true, HAT: "product"},
		{Name: "TS", Input: "ts.inp", Source: "r.xyz", EndSource: "p.xyz", Opt: true, HAT: "ts"},
		{Name: "IRC", Input: "irc.inp", Source: "ts_NEB-TS_converged.xyz", HAT: "irc"},
	}}
}

func TestHATPreflight(t *testing.T) {
	req := hatTestRequest()
	if r := preflight(req); !r.OK {
		t.Fatalf("valid workflow rejected: %v", r.Errors)
	}
	for _, tc := range []struct {
		name string
		edit func(*RunRequest)
	}{
		{"swapped element order", func(r *RunRequest) { r.Files[1].Content = "3\np\nH 1.5 1 0\nC 0 0 0\nC 1.5 0 0\n" }},
		{"wrong NEB endpoint", func(r *RunRequest) { r.Files[4].Content = strings.ReplaceAll(r.Files[4].Content, "p.xyz", "r.xyz") }},
		{"different solvent", func(r *RunRequest) {
			r.Files[4].Content = strings.ReplaceAll(r.Files[4].Content, "CPCM(dmf)", "CPCM(acetonitrile)")
		}},
		{"wrong IRC direction", func(r *RunRequest) {
			r.Files[5].Content = strings.ReplaceAll(r.Files[5].Content, "Direction both", "Direction forward")
		}},
		{"legacy TS basename", func(r *RunRequest) {
			r.Files[5].Content = strings.ReplaceAll(r.Files[5].Content, "ts_NEB-TS_converged.xyz", "ts.xyz")
			r.Stages[3].Source = "ts.xyz"
		}},
		{"different temperature", func(r *RunRequest) { r.Files[4].Content = strings.ReplaceAll(r.Files[4].Content, "298.15", "300") }},
		{"different multiplicity", func(r *RunRequest) { r.Files[4].Content = strings.ReplaceAll(r.Files[4].Content, "0 2", "0 4") }},
		{"no TS frequencies", func(r *RunRequest) {
			r.Files[4].Content = strings.ReplaceAll(r.Files[4].Content, "Freq NEB-TS", "NEB-TS")
		}},
		{"no IRC", func(r *RunRequest) { r.Stages = r.Stages[:3] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := hatTestRequest()
			tc.edit(&r)
			if p := preflight(r); p.OK {
				t.Fatal("invalid HAT workflow accepted")
			}
		})
	}
}

func TestHATReportDoesNotInventMissingEnergy(t *testing.T) {
	result := analyzeHAT(t.TempDir(), hatTestRequest(), nil, nil)
	raw, err := json.Marshal(result)
	if err != nil || strings.Contains(string(raw), "ForwardKcalMol") {
		t.Fatalf("unavailable barrier encoded as a number: %s %v", raw, err)
	}
	r := formatHATResult(result)
	if !strings.Contains(r, "Unavailable") || strings.Contains(r, "Forward ΔG‡ =") {
		t.Fatalf("unexpected report: %s", r)
	}
}

func TestHATMappingDetectsH14H17Swap(t *testing.T) {
	var heavy [14]string
	for i := range heavy {
		heavy[i] = fmt.Sprintf("C %d 0 0", 30+i*4)
	}
	heavy[4], heavy[6], heavy[9] = "C 0 0 0", "C 3 0 0", "C 6 0 0"
	prefix := "19\nfixture\n" + strings.Join(heavy[:], "\n") + "\n"
	tail := "\nC 130 0 0\nC 140 0 0\n"
	r := prefix + "H 0 1 0" + tail + "H 6 1 0\nH 3 1 0\n"
	wrong := prefix + "H 6 1 0" + tail + "H 3 -1 0\nH 3 1 0\n"
	m := hatMapping{H: 14, Donor: 4, Acceptor: 6}
	review, err := reviewHATMapping(r, wrong, m)
	if err != nil || review.Valid || !slices.Equal(review.SwapCandidates, []int{17}) || review.CorrectedProduct == "" {
		t.Fatalf("expected H14/H17 exchange: %v %+v", err, review)
	}
	corrected, err := reviewHATMapping(r, review.CorrectedProduct, m)
	if err != nil || !corrected.Valid || len(corrected.Changes) != 2 {
		t.Fatalf("corrected mapping: %v %+v", err, corrected)
	}
}
