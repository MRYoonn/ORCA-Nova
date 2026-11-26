package main

import (
	"strings"
	"testing"
)

func TestM062XKeywordPaths(t *testing.T) {
	for _, v := range []string{"M06-2X", "m06-2x", "M062X"} {
		if got := orcaFunctionalKeyword(v); got != "M062X" {
			t.Fatalf("%s: %s", v, got)
		}
	}
	if got := orcaFunctionalKeyword("B3LYP"); got != "B3LYP" {
		t.Fatal(got)
	}
	opt := tripOptions{Functional: "M06-2X", Basis: "def2-TZVP", Dispersion: "D4", Solvent: "acetonitrile", Cores: 8, MaxCore: 2000}
	for _, stage := range []string{"constrained", "free", "spin"} {
		k := keywordLine(tripORCAInput(stage, "input.xyz", opt, nil))
		if !k["m062x"] || k["m06-2x"] {
			t.Fatalf("%s: wrong keyword %v", stage, k)
		}
	}
	req := hatTestRequest()
	for i := range req.Files {
		if strings.HasSuffix(req.Files[i].Name, ".inp") {
			req.Files[i].Content = strings.ReplaceAll(req.Files[i].Content, "M062X", "M06-2X")
		}
	}
	if r := preflight(req); !r.OK {
		t.Fatalf("legacy alias should migrate: %+v", r)
	}
	normalized := normalizeORCARequest(req)
	if strings.Contains(normalized.Files[2].Content, "M06-2X") || !strings.Contains(req.Files[2].Content, "M06-2X") {
		t.Fatal("normalization changed source or missed execution keyword")
	}
	raw := "  ! m06-2x def2-TZVP NEB-TS TightSCF  # M06-2X\r\n# M06-2X\r\n%output Print[P_M06-2X] 1 end"
	want := strings.Replace(raw, "m06-2x", "M062X", 1)
	if got := normalizeORCAInput(raw); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got := normalizeORCAInput("! r2SCAN-3c wB97X-D3 def2-QZVPPD"); got != "! r2SCAN-3c wB97X-D3 def2-QZVPPD" {
		t.Fatal(got)
	}
	if !duplicateORCAFunctional("! M06-2X\n! M062X") {
		t.Fatal("duplicate alias")
	}
	if comparisonSettings("! M06-2X def2-TZVP") != comparisonSettings("! M062X def2-TZVP") {
		t.Fatal("aliases not comparable")
	}
}
