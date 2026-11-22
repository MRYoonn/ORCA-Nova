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

func TestWorkflowETAUsesRemainingTimeBounds(t *testing.T) {
	h := newDurationHistory(filepath.Join(t.TempDir(), "config.json"))
	h.observe("known", 600)
	a := &App{history: h}
	now := time.Now()
	j := &Job{State: "running", Started: now.Add(-10 * time.Minute).Format(time.RFC3339), StageCount: 1, stageStarted: now.Add(-10 * time.Minute), stageKey: "known", stageKeys: []string{"known"}}
	v := a.currentETA(j)
	if v.WorkflowETA != v.StageETA || strings.Contains(v.WorkflowETA, "1분 이내 ~ 1분 이내") {
		t.Fatal(v)
	}
	j.stageStarted = now.Add(-20 * time.Minute)
	v = a.currentETA(j)
	if !strings.Contains(v.WorkflowETA, "예측 불안정") {
		t.Fatal(v)
	}
}

func TestBatchETAUsesStageBoundsAndResourceLimits(t *testing.T) {
	h := newDurationHistory(filepath.Join(t.TempDir(), "config.json"))
	for i := 0; i < 3; i++ {
		h.observe("known", 600)
	}
	q := &JobQueue{
		app:    &App{history: h},
		Limits: QueueLimits{Concurrency: 4, CPU: 8, RAMMB: 8000},
	}
	for i := 0; i < 2; i++ {
		q.Jobs = append(q.Jobs, &QueueEntry{
			State: "Running", CPU: 8, RAMMB: 8000,
			Request: RunRequest{Stages: []Stage{{Input: "test.inp"}}},
			Job:     &Job{stageKeys: []string{"known"}},
		})
	}
	if got := q.batchETA(); !strings.HasPrefix(got, "약 14분 ~ 약 28분") {
		t.Fatal(got)
	}
	q.Jobs = q.Jobs[:1]
	q.Jobs[0].ETA.StageElapsedSeconds = 600
	if got := q.batchETA(); !strings.HasPrefix(got, "1분 이내 ~ 약 4분") {
		t.Fatal(got)
	}
	q.Jobs[0].ETA.StageElapsedSeconds = 841
	if got := q.batchETA(); !strings.Contains(got, "예측 불안정") {
		t.Fatal(got)
	}
}
