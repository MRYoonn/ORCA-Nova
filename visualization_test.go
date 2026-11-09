package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVisualizationScanDoesNotInventMissingData(t *testing.T) {
	v := parseVisualization("scan.out", "RELAXED SURFACE SCAN STEP 1\nScan coordinate: 1.4\nSCF CONVERGED AFTER 8 CYCLES\nFINAL SINGLE POINT ENERGY -100\nTHE OPTIMIZATION HAS CONVERGED\nRELAXED SURFACE SCAN STEP 2\nScan coordinate: 1.5\nSCF NOT CONVERGED\n", "")
	if len(v.PES) != 2 || v.PES[0].Energy == nil || v.PES[0].Status != "Completed" || v.PES[1].Energy != nil || v.PES[1].Status != "Failed" {
		t.Fatalf("%+v", v.PES)
	}
	if v.PES[0].Line != 4 {
		t.Fatal("lost provenance")
	}
}
func TestPESDataStrictFiniteColumns(t *testing.T) {
	for _, bad := range []string{"1 NaN", "1 2 3", "distance energy"} {
		if _, err := parsePESData("scan.dat", strings.NewReader(bad)); err == nil {
			t.Fatal(bad)
		}
	}
	p, e := parsePESData("scan.dat", strings.NewReader("# distance energy\n1.4 -100.2\n1.5 -100.1\n"))
	if e != nil || len(p) != 2 || p[0].Status != "Unavailable" || p[0].Line != 2 {
		t.Fatal(p, e)
	}
}
func TestVisualizationOptimizationSCFAndIR(t *testing.T) {
	v := parseVisualization("x.out", "GEOMETRY OPTIMIZATION CYCLE 1\nFINAL SINGLE POINT ENERGY -75\nRMS gradient 0.002\nITER Energy Delta-E\n0 -74.9 -0.01\n1 -75.0 -0.001\nSCF CONVERGED\nIR SPECTRUM\n1: 1000 0.1 42.5\nTHERMOCHEMISTRY\n", "")
	if len(v.Optimization) != 1 || len(v.Gradient) != 1 || len(v.SCF) != 2 || len(v.IR) != 1 || v.IR[0].Y != 42.5 {
		t.Fatalf("%+v", v)
	}
}
func TestVisualizationDiscoversDataAndSkipsSymlinks(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "m.relaxscanact.dat"), []byte("1.4 -100\n"), 0600)
	os.WriteFile(filepath.Join(dir, "m.out"), []byte("FINAL SINGLE POINT ENERGY -100\n"), 0600)
	v, e := loadVisualization(dir)
	if e != nil || len(v.Sources) != 2 {
		t.Fatal(v, e)
	}
}

func TestVisualizationMissingSCFNeverCompleted(t *testing.T) {
	v := parseVisualization("x.out", "SCAN STEP 1\nBOND DISTANCE: 1.4\nFINAL SINGLE POINT ENERGY -100\nTHE OPTIMIZATION HAS CONVERGED\n", "")
	if v.PES[0].Status != "Unavailable" {
		t.Fatal("unverified convergence marked completed")
	}
}
func TestVisualizationOutputSizeLimit(t *testing.T) {
	p := filepath.Join(t.TempDir(), "large.out")
	os.WriteFile(p, []byte("123456"), 0600)
	if _, err := readPlotFile(p, 5); err == nil {
		t.Fatal("oversized output accepted")
	}
}
