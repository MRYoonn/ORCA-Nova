package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestETAUnknownWithoutHistory(t *testing.T) {
	h := newDurationHistory(filepath.Join(t.TempDir(), "config.json"))
	if _, _, ok := h.estimate("unknown"); ok {
		t.Fatal("invented ETA")
	}
	a := &App{history: h}
	v := a.currentETA(&Job{State: "running", Started: time.Now().Add(-5 * time.Minute).Format(time.RFC3339), StageCount: 1, stageStarted: time.Now().Add(-2 * time.Minute), stageKey: "unknown"})
	if !strings.Contains(v.StageETA, "추정 중") || v.StageElapsedSeconds < 110 {
		t.Fatalf("%+v", v)
	}
}
func TestETAHistoryMedianAndLocalPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	h := newDurationHistory(path)
	for _, sec := range []float64{500, 100, 200, 300, 400} {
		h.observe("matching", sec)
	}
	h.observe("another", 10000)
	median, n, ok := newDurationHistory(path).estimate("matching")
	if !ok || median != 300 || n != 5 {
		t.Fatalf("median=%v n=%d ok=%v", median, n, ok)
	}
	if !strings.Contains(rangeEstimate(median, 100, n), "~") {
		t.Fatal("missing uncertainty range")
	}
}
func TestFingerprintIgnoresXYZName(t *testing.T) {
	a := "! PBE0 D4 def2-SVP Opt\n%pal\n nprocs 16\nend\n* xyzfile 0 3 molecule.xyz\n"
	b := strings.Replace(a, "molecule.xyz", "other.xyz", 1)
	if stageFingerprint(a, 24) != stageFingerprint(b, 24) {
		t.Fatal("filename should not change history matching")
	}
	if stageFingerprint(a, 24) == stageFingerprint(a, 25) {
		t.Fatal("molecule size must change history matching")
	}
}
func TestCycleNotProgressPercent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.out")
	if e := os.WriteFile(path, []byte("GEOMETRY OPTIMIZATION CYCLE 4\nGEOMETRY OPTIMIZATION CYCLE 12\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if got := lastOptCycle(path); got != 12 {
		t.Fatal(got)
	}
}
func TestETANoWorkflowPredictionIfAnyUnknown(t *testing.T) {
	h := newDurationHistory(filepath.Join(t.TempDir(), "config.json"))
	h.observe("known", 120)
	a := &App{history: h}
	v := a.currentETA(&Job{State: "running", Started: time.Now().Format(time.RFC3339), StageCount: 2, StageIndex: 0, stageStarted: time.Now().Add(-time.Minute), stageKey: "known", stageKeys: []string{"known", "unknown"}})
	if v.WorkflowETA != "추정 중" {
		t.Fatal(v.WorkflowETA)
	}
	if strings.Contains(v.StageETA, "추정 중") {
		t.Fatal(v.StageETA)
	}
}
