package main

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
)

func (q *JobQueue) batchETA() string {
	if q.Paused {
		return "추정 중 · 대기 작업 보류"
	}
	total, longest := 0.0, 0.0
	active := 0
	for _, e := range q.Jobs {
		if e.State != "Pending" && e.State != "Running" {
			continue
		}
		active++
		if e.Held {
			return "추정 중 · 보류 작업 포함"
		}
		files := map[string]string{}
		atoms := 0
		for _, f := range e.Request.Files {
			files[f.Name] = f.Content
			if f.Name == e.Request.Stages[0].Source {
				atoms, _, _ = parseXYZAtoms(f.Content)
			}
		}
		remain := 0.0
		for i, s := range e.Request.Stages {
			done := false
			if e.Job != nil {
				for _, name := range e.Job.CompletedStages {
					if name == s.Input {
						done = true
					}
				}
			}
			if done {
				continue
			}
			median, _, ok := q.app.history.estimate(stageFingerprint(files[s.Input], atoms))
			if !ok {
				return "추정 중 · 유사 계산 기록 부족"
			}
			elapsed := 0.0
			if e.State == "Running" && e.Job != nil && i == e.Job.StageIndex {
				elapsed = float64(e.ETA.StageElapsedSeconds)
			}
			if elapsed > median*1.75 {
				return "예측 불안정 · 유사 계산보다 오래 진행 중"
			}
			remain += math.Max(0, median-elapsed)
		}
		total += remain
		if remain > longest {
			longest = remain
		}
	}
	if active == 0 {
		return "계산 종료"
	}
	// Lower bound assumes ideal parallelism; upper bound assumes serial work.
	lower := math.Max(longest, total/float64(q.Limits.Concurrency)) * 0.5
	return formatRemain(lower) + " ~ " + formatRemain(total*1.75) + " (과거 기록 기반 범위)"
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
