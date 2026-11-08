package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ETA is based on prior, successfully completed calculations with equivalent
// method/settings, electronic state, core count, and molecular size. Geometry
// iteration counts are NOT percentages of total completion: the number of
// iterations needed for convergence is unknown beforehand.
type durationSample struct {
	Key     string  `json:"key"`
	Seconds float64 `json:"seconds"`
	SavedAt string  `json:"savedAt"`
}
type durationHistory struct {
	mu      sync.Mutex
	path    string
	samples []durationSample
}
type etaView struct {
	ElapsedSeconds      int64  `json:"elapsedSeconds"`
	StageElapsedSeconds int64  `json:"stageElapsedSeconds"`
	StageETA            string `json:"stageEta"`
	WorkflowETA         string `json:"workflowEta"`
	EstimateBasis       string `json:"estimateBasis"`
	OptCycle            int    `json:"optimizationCycle,omitempty"`
}

var optCycleRx = regexp.MustCompile(`(?i)\b(?:GEOMETRY\s+)?OPTIMIZATION\s+CYCLE\s*[:#=]?\s*(\d+)\b`)
var numericHeaderRx = regexp.MustCompile(`(?im)^\s*\*\s*xyz(?:file)?\s+(-?\d+)\s+(\d+)\b`)
var nprocsRx = regexp.MustCompile(`(?im)\b(?:nprocs|nproc)\s+([0-9]+)\b`)

func newDurationHistory(configFile string) *durationHistory {
	h := &durationHistory{path: filepath.Join(filepath.Dir(configFile), "duration_history.json")}
	b, err := os.ReadFile(h.path)
	if err == nil {
		_ = json.Unmarshal(b, &h.samples)
	}
	if len(h.samples) > 500 {
		h.samples = h.samples[len(h.samples)-500:]
	}
	return h
}
func stageFingerprint(input string, atoms int) string {
	if atoms <= 0 {
		return ""
	}
	var kws []string
	for _, line := range strings.Split(input, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "!") {
			kws = append(kws, strings.Fields(strings.ToLower(strings.TrimSpace(strings.TrimPrefix(t, "!"))))...)
		}
	}
	if len(kws) == 0 {
		return ""
	}
	sort.Strings(kws)
	charge, mult := "?", "?"
	if m := numericHeaderRx.FindStringSubmatch(input); len(m) == 3 {
		charge, mult = m[1], m[2]
	}
	cores := "1"
	if m := nprocsRx.FindStringSubmatch(input); len(m) == 2 {
		cores = m[1]
	}
	// Include solvent, TDDFT roots, and other block settings; normalize the
	// local input filename/XYZ reference, not the molecular coordinates.
	var extra []string
	for _, line := range strings.Split(input, "\n") {
		l := strings.ToLower(strings.TrimSpace(line))
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "!") || strings.HasPrefix(l, "* xyz") || strings.HasPrefix(l, "%pal") || strings.HasPrefix(l, "nprocs") || strings.HasPrefix(l, "%maxcore") || strings.HasPrefix(l, "end") {
			continue
		}
		extra = append(extra, strings.Join(strings.Fields(l), " "))
	}
	payload := fmt.Sprintf("atoms=%d|charge=%s|mult=%s|cores=%s|kw=%s|extra=%s", atoms, charge, mult, cores, strings.Join(kws, " "), strings.Join(extra, ";"))
	s := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(s[:])[:24]
}
func xyzAtomCount(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	lines := strings.SplitN(string(b), "\n", 2)
	v, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil || v <= 0 || v > 100000 {
		return 0
	}
	return v
}
func signatureForStage(folder string, stage Stage, fallbackAtoms int) string {
	b, err := os.ReadFile(filepath.Join(folder, stage.Input))
	if err != nil {
		return ""
	}
	atoms := xyzAtomCount(filepath.Join(folder, stage.Source))
	if atoms == 0 {
		atoms = fallbackAtoms
	}
	return stageFingerprint(string(b), atoms)
}
func (h *durationHistory) observe(key string, seconds float64) {
	if h == nil || key == "" || seconds < 0.1 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.samples = append(h.samples, durationSample{Key: key, Seconds: seconds, SavedAt: time.Now().Format(time.RFC3339)})
	if len(h.samples) > 500 {
		h.samples = h.samples[len(h.samples)-500:]
	}
	b, err := json.MarshalIndent(h.samples, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(h.path), 0700); err != nil {
		return
	}
	tmp := h.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err == nil {
		_ = os.Rename(tmp, h.path)
	}
}
func (h *durationHistory) estimate(key string) (median float64, samples int, ok bool) {
	if h == nil || key == "" {
		return 0, 0, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	matches := []float64{}
	for i := len(h.samples) - 1; i >= 0 && len(matches) < 7; i-- {
		if h.samples[i].Key == key && h.samples[i].Seconds > 0 {
			matches = append(matches, h.samples[i].Seconds)
		}
	}
	if len(matches) == 0 {
		return 0, 0, false
	}
	sort.Float64s(matches)
	mid := len(matches) / 2
	if len(matches)%2 == 1 {
		median = matches[mid]
	} else {
		median = (matches[mid-1] + matches[mid]) / 2
	}
	return median, len(matches), true
}
func formatRemain(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	minutes := int64(math.Round(seconds / 60))
	if minutes <= 0 {
		return "1분 이내"
	}
	if minutes < 60 {
		return fmt.Sprintf("약 %d분", minutes)
	}
	return fmt.Sprintf("약 %d시간 %d분", minutes/60, minutes%60)
}
func rangeEstimate(median, elapsed float64, n int) string {
	// Wide uncertainty for 1-2 samples; tighter only after several comparable
	// successful jobs. Never pretend the forecast is an accurate deadline.
	lower, upper := 0.5, 1.75
	if n >= 3 {
		lower, upper = 0.7, 1.4
	}
	if elapsed > median*upper {
		return "유사 계산보다 오래 진행 중 · 재추정 어려움"
	}
	lo := math.Max(0, median*lower-elapsed)
	hi := math.Max(0, median*upper-elapsed)
	return formatRemain(lo) + " ~ " + formatRemain(hi)
}
func lastOptCycle(out string) int {
	if out == "" {
		return 0
	}
	contents := tail(out, 200000)
	matches := optCycleRx.FindAllStringSubmatch(contents, -1)
	if len(matches) == 0 {
		return 0
	}
	v, _ := strconv.Atoi(matches[len(matches)-1][1])
	return v
}
func (a *App) currentETA(job *Job) etaView {
	result := etaView{StageETA: "추정 중 · 유사 계산 기록 없음", WorkflowETA: "추정 중", EstimateBasis: "과거 완료된 유사 계산 기준 · 정확한 종료 시각은 보장되지 않음"}
	if job == nil {
		return result
	}
	started, err := time.Parse(time.RFC3339, job.Started)
	if err == nil {
		result.ElapsedSeconds = int64(math.Max(0, time.Since(started).Seconds()))
	}
	if job.State != "running" {
		result.StageETA = "계산 종료"
		result.WorkflowETA = "계산 종료"
		return result
	}
	if !job.stageStarted.IsZero() {
		result.StageElapsedSeconds = int64(math.Max(0, time.Since(job.stageStarted).Seconds()))
	}
	result.OptCycle = lastOptCycle(job.OutFile)
	if median, n, ok := a.history.estimate(job.stageKey); ok {
		result.StageETA = rangeEstimate(median, float64(result.StageElapsedSeconds), n)
		result.EstimateBasis = fmt.Sprintf("유사 설정의 완료 기록 %d개 기반 · 실제 수렴에 따라 변동", n)
	}
	if len(job.stageKeys) == 0 {
		return result
	}
	sum := float64(0)
	minSamples := 1000
	for i, key := range job.stageKeys {
		if i < job.StageIndex {
			continue
		}
		median, n, ok := a.history.estimate(key)
		if !ok {
			return result
		}
		if n < minSamples {
			minSamples = n
		}
		if i == job.StageIndex {
			sum += math.Max(0, median-float64(result.StageElapsedSeconds))
		} else {
			sum += median
		}
	}
	// Only show full-workflow ETA if every remaining stage has comparable history.
	result.WorkflowETA = rangeEstimate(sum, 0, minSamples)
	return result
}
