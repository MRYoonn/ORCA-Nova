package main

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func (q *JobQueue) batchETA() string {
	if q.Paused {
		return "추정 중 · 대기 작업 보류"
	}
	totalLow, totalHigh, longest := 0.0, 0.0, 0.0
	cpuSeconds, ramSeconds := 0.0, 0.0
	active := 0
	for _, e := range q.Jobs {
		if e.State != "Pending" && e.State != "Running" {
			continue
		}
		active++
		if e.Held {
			return "추정 중 · 보류 작업 포함"
		}
		if len(e.Request.Stages) == 0 {
			return "추정 중 · 실행 단계 정보 없음"
		}
		files := make(map[string]string, len(e.Request.Files))
		for _, f := range e.Request.Files {
			files[f.Name] = f.Content
		}
		atoms := xyzHeaderAtomCount(files[e.Request.Stages[0].Source])
		geometries := make(map[string]int)
		low := 0.0
		for i, stage := range e.Request.Stages {
			stageAtoms := xyzHeaderAtomCount(files[stage.Source])
			if stageAtoms == 0 {
				stageAtoms = geometries[stage.Source]
			}
			if generated := stageGeometry(stage); generated != "" {
				geometries[generated] = stageAtoms
			}
			if e.Job != nil && slices.Contains(e.Job.CompletedStages, stage.Input) {
				continue
			}
			if stageAtoms == 0 {
				stageAtoms = atoms
			}
			key := stageFingerprint(files[stage.Input], stageAtoms)
			if e.Job != nil && i < len(e.Job.stageKeys) {
				key = e.Job.stageKeys[i]
			}
			median, samples, ok := q.app.history.estimate(key)
			if !ok {
				return "추정 중 · 유사 계산 기록 부족"
			}
			elapsed := 0.0
			if e.State == "Running" && e.Job != nil && i == e.Job.StageIndex {
				elapsed = float64(e.ETA.StageElapsedSeconds)
			}
			stageLow, stageHigh, overdue := remainingBounds(median, elapsed, samples)
			if overdue {
				return "예측 불안정 · 유사 계산보다 오래 진행 중"
			}
			low += stageLow
			totalHigh += stageHigh
		}
		totalLow += low
		longest = math.Max(longest, low)
		cpuSeconds += low * float64(e.CPU)
		ramSeconds += low * float64(e.RAMMB)
	}
	if active == 0 {
		return "계산 종료"
	}
	// Compare the longest job and the work required under each resource limit.
	// The upper estimate assumes serial execution without external waits.
	lower := math.Max(longest, totalLow/float64(max(1, q.Limits.Concurrency)))
	if q.Limits.CPU > 0 {
		lower = math.Max(lower, cpuSeconds/float64(q.Limits.CPU))
	}
	if q.Limits.RAMMB > 0 {
		lower = math.Max(lower, ramSeconds/float64(q.Limits.RAMMB))
	}
	return formatRemain(math.Min(lower, totalHigh)) + " ~ " + formatRemain(totalHigh) + " (과거 기록 기반 범위)"
}
func (q *JobQueue) writeBatchLocked() {
	q.app.mu.Lock()
	root := q.app.cfg.WorkRoot
	q.app.mu.Unlock()
	if root == "" {
		return
	}
	var csv bytes.Buffer
	q.exportLocked(&csv)
	var b strings.Builder
	q.exportHumanTextLocked(&b)
	for name, data := range map[string][]byte{"batch_results.csv": csv.Bytes(), "batch_summary.txt": []byte(b.String())} {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path+".tmp", data, 0600); err != nil {
			q.ExportError = err.Error()
			return
		}
		if err := os.Rename(path+".tmp", path); err != nil {
			q.ExportError = err.Error()
			return
		}
	}
}
