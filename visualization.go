package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type PlotPoint struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Source string  `json:"source"`
	Line   int     `json:"line"`
}
type PESPoint struct {
	Step       int      `json:"step"`
	Coordinate *float64 `json:"coordinate"`
	Energy     *float64 `json:"energyEh"`
	Status     string   `json:"status"`
	Source     string   `json:"source"`
	Line       int      `json:"line"`
	XYZ        string   `json:"xyz,omitempty"`
	Analysis   *ScanStepAnalysis `json:"analysis,omitempty"`
}
// Step analysis is present only for population rows actually printed inside
// the matching PES scan step; missing atomic data remains unavailable.
type ScanStepAnalysis struct {
	Populations map[string][]AtomicPopulation `json:"Populations"`
}
type VisualizationSource struct {
	RawOutput    string       `json:"rawOutput,omitempty"`
	Input        string       `json:"input,omitempty"`
	Name         string       `json:"name"`
	Analysis     OrcaAnalysis `json:"analysis"`
	PES          []PESPoint   `json:"pes"`
	Optimization []PlotPoint  `json:"optimization"`
	Gradient     []PlotPoint  `json:"gradient"`
	SCF          []PlotPoint  `json:"scf"`
	SCFDelta     []PlotPoint  `json:"scfDelta"`
	IR           []PlotPoint  `json:"ir"`
	Warnings     []string     `json:"warnings"`
}
type Visualization struct {
	Sources  []VisualizationSource `json:"sources"`
	Units    map[string]string     `json:"units"`
	Warnings []string              `json:"warnings"`
}

var scanXYZRX = regexp.MustCompile(`(?i)(?:XYZ FILE|GEOMETRY FILE)\s*[:=]\s*([A-Za-z0-9_.-]+\.xyz)\b`)
var scanStepRX = regexp.MustCompile(`(?i)(?:RELAXED\s+SURFACE\s+SCAN\s+STEP|SCAN\s+STEP)\s*[:#]?\s*(\d+)`)
var scanCoordRX = regexp.MustCompile(`(?i)(?:SCAN COORDINATE|CURRENT VALUE|BOND DISTANCE)\s*[:=]\s*(` + orcaNumber + `)`)
var cycleRX = regexp.MustCompile(`(?i)GEOMETRY OPTIMIZATION CYCLE\s*(\d+)`)
var rmsGradientRX = regexp.MustCompile(`(?i)RMS gradient\s+(` + orcaNumber + `)`)
var numericRowRX = regexp.MustCompile(`^\s*(\d+)\s+(` + orcaNumber + `)\s+(` + orcaNumber + `)`)
var irRowRX = regexp.MustCompile(`^\s*(\d+)\s*:\s*(` + orcaNumber + `)\s+(` + orcaNumber + `)\s+(` + orcaNumber + `)`)

func finiteNumber(s string) (float64, bool) {
	v, e := number(s)
	return v, e == nil && !math.IsNaN(v) && !math.IsInf(v, 0)
}
func parseVisualization(name, output, inp string) VisualizationSource {
	a, err := ParseOrcaOutput(strings.NewReader(output), name, inp)
	v := VisualizationSource{Name: name, Analysis: a, RawOutput: output, Input: inp, PES: []PESPoint{}, Optimization: []PlotPoint{}, Gradient: []PlotPoint{}, SCF: []PlotPoint{}, SCFDelta: []PlotPoint{}, IR: []PlotPoint{}, Warnings: []string{}}
	if err != nil {
		v.Warnings = append(v.Warnings, err.Error())
	}
	scanSCF := []bool{}
	scan := -1
	spinSection := ""
	spinRows := map[string][]AtomicPopulation{}
	cycle := 0
	scfTable := false
	irTable := false
	scfBlock := 0
	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Buffer(make([]byte, 4096), 4<<20)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		s := scanner.Text()
		u := strings.ToUpper(s)
		if m := scanStepRX.FindStringSubmatch(s); m != nil {
			step, _ := strconv.Atoi(m[1])
			v.PES = append(v.PES, PESPoint{Step: step, Status: "Unavailable", Source: name, Line: lineNo})
			scanSCF = append(scanSCF, false)
			scan = len(v.PES) - 1
			spinRows = map[string][]AtomicPopulation{}
			spinSection = ""
		}
		if scan >= 0 {
			// Capture atomic spin tables only within this explicit scan step.
			// OUT files that print only one final population table must not
			// fabricate values for all earlier points.
			switch {
			case strings.Contains(u, "MULLIKEN ATOMIC CHARGES"):
				spinSection = "Mulliken"
				spinRows[spinSection] = nil
			case strings.Contains(u, "LOEWDIN ATOMIC CHARGES"), strings.Contains(u, "LÖWDIN ATOMIC CHARGES"):
				spinSection = "Loewdin"
				spinRows[spinSection] = nil
			case strings.Contains(u, "HIRSHFELD ATOMIC CHARGES"), strings.Contains(u, "HIRSHFELD ANALYSIS"):
				spinSection = "Hirshfeld"
				spinRows[spinSection] = nil
			case strings.Contains(u, "VIBRATIONAL FREQUENCIES"),
				strings.Contains(u, "ORBITAL ENERGIES"),
				strings.Contains(u, "NORMAL MODES"),
				strings.Contains(u, "IR SPECTRUM"),
				strings.Contains(u, "MAYER POPULATION ANALYSIS"):
				spinSection = ""
			}
			if spinSection != "" {
				m := atomRx.FindStringSubmatch(s)
				if spinSection == "Hirshfeld" && len(m) == 0 {
					m = hirshfeldAtomRx.FindStringSubmatch(s)
				}
				if len(m) > 4 && m[4] != "" && len(spinRows[spinSection]) < 1024 {
					i, _ := strconv.Atoi(m[1])
					q, qErr := number(m[3])
					spin, spinErr := number(m[4])
					if qErr == nil && spinErr == nil && !math.IsNaN(spin) && !math.IsInf(spin, 0) {
						spinRows[spinSection] = append(spinRows[spinSection], AtomicPopulation{Index: i, Element: m[2], Charge: q, Spin: &spin})
					}
				}
			}
			p := &v.PES[scan]
			if m := scanXYZRX.FindStringSubmatch(s); m != nil && safeName(m[1]) {
				p.XYZ = m[1]
			}
			if m := scanCoordRX.FindStringSubmatch(s); m != nil {
				if x, ok := finiteNumber(m[1]); ok {
					p.Coordinate = &x
				}
			}
			if m := finalEnergyRx.FindStringSubmatch(s); m != nil {
				if x, ok := finiteNumber(m[1]); ok {
					p.Energy = &x
					p.Line = lineNo
				}
			}
			if strings.Contains(u, "SCF CONVERGED") {
				scanSCF[scan] = true
			}
			if strings.Contains(u, "THE OPTIMIZATION HAS CONVERGED") && p.Status != "Failed" {
				p.Status = "Completed"
			}
			if strings.Contains(u, "DID NOT CONVERGE") || strings.Contains(u, "NOT CONVERGED") || strings.Contains(u, "ERROR TERMINATION") {
				p.Status = "Failed"
			}
			if len(spinRows) > 0 {
				pops := map[string][]AtomicPopulation{}
				for scheme, rows := range spinRows {
					if len(rows) > 0 { pops[scheme] = append([]AtomicPopulation(nil), rows...) }
				}
				if len(pops) > 0 { p.Analysis = &ScanStepAnalysis{Populations: pops} }
			}
		}
		if m := cycleRX.FindStringSubmatch(s); m != nil {
			cycle, _ = strconv.Atoi(m[1])
		}
		if m := finalEnergyRx.FindStringSubmatch(s); m != nil && cycle > 0 {
			if y, ok := finiteNumber(m[1]); ok {
				v.Optimization = append(v.Optimization, PlotPoint{float64(cycle), y, name, lineNo})
			}
		}
		if m := rmsGradientRX.FindStringSubmatch(s); m != nil && cycle > 0 {
			if y, ok := finiteNumber(m[1]); ok {
				v.Gradient = append(v.Gradient, PlotPoint{float64(cycle), y, name, lineNo})
			}
		}
		if strings.Contains(u, "ITER") && strings.Contains(u, "ENERGY") && strings.Contains(u, "DELTA-E") {
			scfTable = true
			scfBlock++
			continue
		}
		if scfTable {
			if m := numericRowRX.FindStringSubmatch(s); m != nil {
				x, _ := strconv.Atoi(m[1])
				y, ok := finiteNumber(m[2])
				d, ok2 := finiteNumber(m[3])
				if ok && ok2 {
					source := fmt.Sprintf("%s (SCF block %d)", name, scfBlock)
					v.SCF = append(v.SCF, PlotPoint{float64(x), y, source, lineNo})
					v.SCFDelta = append(v.SCFDelta, PlotPoint{float64(x), d, source, lineNo})
				}
			} else if strings.TrimSpace(s) == "" || strings.Contains(u, "SCF CONVERGED") {
				scfTable = false
			}
		}
		if strings.Contains(u, "IR SPECTRUM") {
			irTable = true
			continue
		}
		if irTable {
			if m := irRowRX.FindStringSubmatch(s); m != nil {
				x, ok := finiteNumber(m[2])
				y, ok2 := finiteNumber(m[4])
				if ok && ok2 {
					v.IR = append(v.IR, PlotPoint{x, y, name, lineNo})
				}
			} else if strings.Contains(u, "RAMAN") || strings.Contains(u, "THERMOCHEMISTRY") {
				irTable = false
			}
		}
	}
	if err := scanner.Err(); err != nil {
		v.Warnings = append(v.Warnings, err.Error())
	}
	for i := range v.PES {
		p := &v.PES[i]
		if (p.Energy == nil || !scanSCF[i]) && p.Status == "Completed" {
			p.Status = "Unavailable"
		}
		if p.Analysis == nil {
			v.Warnings = append(v.Warnings, fmt.Sprintf("Scan step %d: 해당 단계의 Atomic Spin Population이 없어 Spin 곡선에서 제외됩니다.", p.Step))
		}
		if p.Coordinate == nil {
			v.Warnings = append(v.Warnings, fmt.Sprintf("Scan step %d: 실제 Bond Distance를 인식하지 못했습니다. 점을 생성하지 않습니다.", p.Step))
		}
	}
	if len(v.SCF) > 0 {
		v.Warnings = append(v.Warnings, "SCF iteration 번호는 SCF block마다 다시 시작합니다. source의 block 번호를 함께 확인하세요.")
	}
	return v
}
func parsePESData(name string, r io.Reader) ([]PESPoint, error) {
	points := []PESPoint{}
	s := bufio.NewScanner(r)
	line := 0
	for s.Scan() {
		line++
		text := strings.TrimSpace(s.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		f := strings.Fields(text)
		if len(f) != 2 {
			return nil, fmt.Errorf("%s:%d: Scan DAT는 coordinate와 Electronic Energy 두 열이어야 합니다", name, line)
		}
		x, ok := finiteNumber(f[0])
		y, ok2 := finiteNumber(f[1])
		if !ok || !ok2 {
			return nil, fmt.Errorf("%s:%d: 유한한 숫자가 필요합니다", name, line)
		}
		points = append(points, PESPoint{Step: len(points) + 1, Coordinate: &x, Energy: &y, Status: "Unavailable", Source: name, Line: line})
	}
	return points, s.Err()
}
func newVisualization() Visualization {
	return Visualization{Sources: []VisualizationSource{}, Units: map[string]string{"pes.coordinate": "Å only when input scan is a bond distance; verify input", "pes.energyEh": "Eh", "optimization.x": "cycle", "optimization.y": "Eh", "gradient.y": "Eh/Bohr", "scf.x": "iteration within block", "scf.y": "Eh", "scfDelta.y": "Eh", "ir.x": "cm^-1", "ir.y": "km/mol"}, Warnings: []string{"전자상태·Method·Geometry가 다른 에너지는 검증 없이 비교하지 마세요. Spin Population은 선택성을 증명하지 않습니다."}}
}
func loadVisualization(folder string) (Visualization, error) {
	v := newVisualization()
	entries, err := os.ReadDir(folder)
	if err != nil {
		return v, err
	}
	budget := int64(32 << 20)
	for _, e := range entries {
		if e.IsDir() || e.Type()&os.ModeSymlink != 0 {
			continue
		}
		if budget <= 0 {
			v.Warnings = append(v.Warnings, "그래프 전체 분석 32MB 제한에 도달했습니다. 나머지 원본은 보존됩니다.")
			break
		}
		name := e.Name()
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".out") {
			f, err := readPlotFile(filepath.Join(folder, name), 8<<20)
			if err != nil {
				v.Warnings = append(v.Warnings, err.Error())
				continue
			}
			inp, _ := readPlotFile(filepath.Join(folder, strings.TrimSuffix(name, filepath.Ext(name))+".inp"), 1<<20)
			budget -= int64(len(f) + len(inp))
			src := parseVisualization(name, string(f), string(inp))
			for i := range src.PES {
				if src.PES[i].XYZ != "" {
					info, err := os.Lstat(filepath.Join(folder, src.PES[i].XYZ))
					if err != nil || !info.Mode().IsRegular() {
						src.PES[i].XYZ = ""
					}
				}
			}
			v.Sources = append(v.Sources, src)
		}
		if strings.HasSuffix(lower, ".dat") && (strings.Contains(lower, "relaxscan") || strings.Contains(lower, "scanact")) {
			f, err := readPlotFile(filepath.Join(folder, name), 8<<20)
			if err != nil {
				continue
			}
			points, err := parsePESData(name, strings.NewReader(string(f)))
			budget -= int64(len(f))
			if err != nil {
				v.Warnings = append(v.Warnings, err.Error())
				continue
			}
			v.Sources = append(v.Sources, VisualizationSource{Name: name, PES: points, Warnings: []string{"DAT에는 수렴 상태와 좌표 종류가 없습니다. 입력과 단계별 OUT을 확인하세요. 실패 지점이 생략됐을 수 있으므로 행 번호를 실제 step으로 확정하지 마세요."}})
		}
	}
	return v, nil
}
func (a *App) visualizationRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var x struct {
			Name   string `json:"name"`
			Output string `json:"output"`
			Input  string `json:"input"`
		}
		if json.NewDecoder(io.LimitReader(r.Body, 32<<20)).Decode(&x) != nil {
			fail(w, 400, "OUT을 읽지 못했습니다")
			return
		}
		if len(x.Output) > 8<<20 || len(x.Input) > 1<<20 {
			fail(w, 400, "OUT은 8MB, INP는 1MB 이하로 불러오세요. 원본은 변경되지 않습니다.")
			return
		}
		if len(x.Output) == 0 {
			fail(w, 400, "실제 OUT 데이터가 필요합니다")
			return
		}
		v := newVisualization()
		v.Sources = append(v.Sources, parseVisualization(filepath.Base(x.Name), x.Output, x.Input))
		respond(w, 200, v)
		return
	}
	if r.Method != http.MethodGet {
		fail(w, 405, "Method not allowed")
		return
	}
	folder := ""
	id := r.URL.Query().Get("id")
	if id != "" && a.queue != nil {
		a.queue.mu.Lock()
		for _, e := range a.queue.Jobs {
			if e.ID == id && e.Job != nil {
				folder = e.Job.Folder
			}
		}
		a.queue.mu.Unlock()
	} else if id == "" {
		a.mu.Lock()
		if a.job != nil {
			folder = a.job.Folder
		}
		a.mu.Unlock()
	}
	if folder == "" {
		fail(w, 404, "계산 결과가 없습니다")
		return
	}
	if r.URL.Path == "/api/visualization/xyz" {
		name := r.URL.Query().Get("file")
		if !safeName(name) || !strings.EqualFold(filepath.Ext(name), ".xyz") {
			fail(w, 400, "XYZ 파일명이 올바르지 않습니다")
			return
		}
		path := filepath.Join(folder, name)
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			fail(w, 404, "XYZ 파일이 없습니다")
			return
		}
		b, err := readPlotFile(path, 8<<20)
		if err != nil {
			fail(w, 400, err.Error())
			return
		}
		if _, _, err = parseXYZAtoms(string(b)); err != nil {
			fail(w, 400, err.Error())
			return
		}
		w.Header().Set("Content-Type", "chemical/x-xyz")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
		w.Write(b)
		return
	}
	v, err := loadVisualization(folder)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	respond(w, 200, v)
}

func readPlotFile(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("%s: 그래프 분석 크기 제한 %dMB 초과; 원본 파일은 보존됩니다", filepath.Base(path), limit>>20)
	}
	return b, nil
}
