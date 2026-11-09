package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
)

// Profiles never infer molecular identity or reaction-state roles from filenames.
// The user explicitly assigns calculated structures and confirms stoichiometry.
type ScientificProfileRequest struct {
	JobID     string            `json:"jobId"`
	Jobs      map[string]string `json:"jobs,omitempty"` // role -> existing queue Job ID, never a filesystem path
	Kind      string            `json:"kind"`           // adiabatic or reaction
	Energy    string            `json:"energy"`         // Electronic or Gibbs
	Sources   map[string]string `json:"sources"`
	Confirmed bool              `json:"confirmed"`
}
type ScientificProfilePoint struct {
	Label    string  `json:"label"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"` // relative energy in kJ/mol
	EnergyEh float64 `json:"energyEh"`
	Source   string  `json:"source"`
}
type ScientificProfileResult struct {
	Kind       string                   `json:"kind"`
	Energy     string                   `json:"energy"`
	Units      string                   `json:"units"`
	Points     []ScientificProfilePoint `json:"points"`
	GapKJ      *float64                 `json:"gapKJ,omitempty"`
	BarrierKJ  *float64                 `json:"barrierKJ,omitempty"`
	ReactionKJ *float64                 `json:"reactionKJ,omitempty"`
	Warnings   []string                 `json:"warnings"`
}

func canonicalMethod(a OrcaAnalysis) string {
	// Only compare the computational level and solvent keyword, not Opt/Freq/OptTS
	// or spin multiplicity. Different method/basis/solvent means no automatic gap.
	drop := map[string]bool{"opt": true, "optts": true, "freq": true, "numfreq": true,
		"sp": true, "tightopt": true, "ukS": true, "uks": true, "rhf": true, "uhf": true,
		"noautostart": true, "printmos": true, "printgap": true}
	var tokens []string
	for _, tok := range strings.Fields(strings.ToLower(strings.TrimSpace(a.Method))) {
		if !drop[tok] {
			tokens = append(tokens, tok)
		}
	}
	return strings.Join(tokens, " ")
}
func profileEnergy(a OrcaAnalysis, mode string, isTS bool) (float64, error) {
	if !a.Normal || !a.SCFConverged || !a.Optimized {
		return 0, fmt.Errorf("%s: 정상 종료·SCF 및 Geometry Optimization 수렴이 모두 확인되어야 합니다", a.Filename)
	}
	if mode == "Gibbs" {
		if a.Gibbs == nil || a.Temperature == nil || a.ImaginaryCount == nil {
			return 0, fmt.Errorf("%s: Gibbs/Temperature/Frequency 정보가 완전하지 않습니다", a.Filename)
		}
		expected := 0
		if isTS {
			expected = 1
		}
		if *a.ImaginaryCount != expected {
			return 0, fmt.Errorf("%s: 허수 진동수 %d개 (기대 %d개)", a.Filename, *a.ImaginaryCount, expected)
		}
		if !isFinite(*a.Gibbs) {
			return 0, fmt.Errorf("%s: 유효하지 않은 Gibbs 값", a.Filename)
		}
		return *a.Gibbs, nil
	}
	if !isTS && a.ImaginaryCount != nil && *a.ImaginaryCount != 0 {
		return 0, fmt.Errorf("%s: Minimum 구조에 허수 진동수가 있습니다", a.Filename)
	}
	if a.Electronic == nil || !isFinite(*a.Electronic) {
		return 0, fmt.Errorf("%s: 전자 에너지가 없습니다", a.Filename)
	}
	if isTS && (a.ImaginaryCount == nil || *a.ImaginaryCount != 1) {
		return 0, fmt.Errorf("%s: TS는 정확히 하나의 허수 진동수로 검증되어야 합니다", a.Filename)
	}
	return *a.Electronic, nil
}
func isFinite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func scientificProfile(req ScientificProfileRequest, sources map[string]OrcaAnalysis) (ScientificProfileResult, error) {
	result := ScientificProfileResult{Kind: req.Kind, Energy: req.Energy, Units: "kJ/mol", Points: []ScientificProfilePoint{}, Warnings: []string{}}
	if !req.Confirmed {
		return result, fmt.Errorf("사용자가 동일한 원자 구성/반응식과 계산 조건을 확인해야 합니다")
	}
	if req.Kind != "adiabatic" && req.Kind != "reaction" {
		return result, fmt.Errorf("지원하지 않는 Profile 종류")
	}
	if req.Energy != "Electronic" && req.Energy != "Gibbs" {
		return result, fmt.Errorf("Energy는 Electronic 또는 Gibbs로 선택하세요")
	}
	roles := []string{"S0", "T1"}
	if req.Kind == "reaction" {
		roles = []string{"Reactant", "TS", "Product"}
	}
	ordered := make([]OrcaAnalysis, 0, len(roles))
	seen := map[string]bool{}
	for _, role := range roles {
		filename := req.Sources[role]
		if filename == "" || seen[filename] {
			return result, fmt.Errorf("%s: 겹치지 않는 결과 파일을 지정하세요", role)
		}
		seen[filename] = true
		a, ok := sources[filename]
		if !ok {
			return result, fmt.Errorf("%s: 해당 결과 파일을 찾지 못했습니다", role)
		}
		if a.Method == "" || a.Geometry == "" || canonicalMethod(a) == "" {
			return result, fmt.Errorf("%s: 원본 INP의 Method/Geometry 정보가 필요합니다", filename)
		}
		ordered = append(ordered, a)
	}
	ref := ordered[0]
	for _, a := range ordered {
		lower := strings.ToLower(a.Method + "\n" + a.Settings)
		if strings.Contains(lower, "tddft") || strings.Contains(lower, "td-dft") || strings.Contains(lower, "iroot") {
			return result, fmt.Errorf("TD-DFT/IRoot 결과는 UKS Adiabatic 또는 기본 Reaction Profile로 자동 해석하지 않습니다")
		}
	}
	for i, a := range ordered {
		if canonicalMethod(a) != canonicalMethod(ref) || profileBlocks(a.Settings) != profileBlocks(ref.Settings) {
			return result, fmt.Errorf("%s: Functional/Basis/Dispersion/Solvent/SCF 설정이 다른 결과는 직접 비교하지 않습니다", roles[i])
		}
		if a.Charge != ref.Charge {
			return result, fmt.Errorf("%s: Charge가 서로 다릅니다", roles[i])
		}
		if req.Energy == "Gibbs" && (ref.Temperature == nil || a.Temperature == nil || math.Abs(*a.Temperature-*ref.Temperature) > 0.5) {
			return result, fmt.Errorf("%s: Gibbs 계산의 온도를 일치시켜야 합니다", roles[i])
		}
	}
	if req.Kind == "adiabatic" {
		if ordered[0].Multiplicity != 1 || ordered[1].Multiplicity != 3 {
			return result, fmt.Errorf("Adiabatic S0/T1 비교에는 Singlet(1) 및 Triplet(3) 최적화 계산이 필요합니다")
		}
		result.Warnings = append(result.Warnings,
			"두 구조의 동일 분자 여부, Triplet 상태 정체성 및 UKS <S²>는 사용자가 검증해야 합니다.",
			"이 값은 최적화 구조 간 Adiabatic 차이이며 S0 구조의 TD-DFT Vertical T1과 다릅니다.")
	} else {
		if ordered[0].Multiplicity != ordered[1].Multiplicity || ordered[0].Multiplicity != ordered[2].Multiplicity {
			return result, fmt.Errorf("Reaction Profile은 동일한 Spin Surface의 구조만 자동 비교합니다")
		}
		if ordered[1].ImaginaryCount == nil || *ordered[1].ImaginaryCount != 1 {
			return result, fmt.Errorf("TS 검증용 Frequency (허수 모드 정확히 하나)가 필요합니다")
		}
		result.Warnings = append(result.Warnings,
			"원자 조성/반응식/표준상태/TS 연결성(IRC)과 계산 수준 일치는 사용자 확인 사항입니다.",
			"다분자 반응물 및 생성물의 에너지 합산은 이 3-구조 Profile에 포함되지 않습니다.")
	}
	values := make([]float64, len(ordered))
	for i, a := range ordered {
		n, e := profileEnergy(a, req.Energy, req.Kind == "reaction" && i == 1)
		if e != nil {
			return result, e
		}
		for _, warning := range a.Warnings {
			result.Warnings = append(result.Warnings, roles[i]+": "+warning)
		}
		if a.S2 != nil {
			spin := float64(a.Multiplicity-1) / 2
			expected := spin * (spin + 1)
			if math.Abs(*a.S2-expected) > 0.2 {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: Spin Contamination 검토 필요 (<S²>=%.4f, 이상값 %.4f)", roles[i], *a.S2, expected))
			}
		}
		values[i] = n
		result.Points = append(result.Points, ScientificProfilePoint{
			Label: roles[i], X: float64(i + 1), Y: (n - values[0]) * ehToKJ,
			EnergyEh: n, Source: req.Sources[roles[i]],
		})
	}
	if req.Kind == "adiabatic" {
		gap := (values[1] - values[0]) * ehToKJ
		result.GapKJ = &gap
	} else {
		barrier := (values[1] - values[0]) * ehToKJ
		reaction := (values[2] - values[0]) * ehToKJ
		result.BarrierKJ = &barrier
		result.ReactionKJ = &reaction
	}
	return result, nil
}

// profileFolder resolves IDs against this application's own queue; it never
// accepts caller-supplied filesystem paths. An empty ID uses the active job.
func (a *App) profileFolder(id string) (string, error) {
	if id == "" {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.job == nil || a.job.Folder == "" {
			return "", fmt.Errorf("현재 단일 Job 계산 결과가 없습니다")
		}
		return a.job.Folder, nil
	}
	if a.queue == nil {
		return "", fmt.Errorf("등록된 Batch Job이 없습니다")
	}
	a.queue.mu.Lock()
	defer a.queue.mu.Unlock()
	for _, entry := range a.queue.Jobs {
		if entry != nil && entry.ID == id && entry.Job != nil && entry.Job.Folder != "" {
			return entry.Job.Folder, nil
		}
	}
	return "", fmt.Errorf("Batch Job %q을 찾지 못했습니다", id)
}

// xyzFormula reads the exact composition, not merely atom count or total Z.
// This catches iso-electronic but different-element geometries before a gap
// can be presented as physically comparable.
func xyzFormula(raw string) (string, error) {
	count, _, err := parseXYZAtoms(raw)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	atoms := map[string]int{}
	for i := 0; i < count; i++ {
		fields := strings.Fields(lines[i+2])
		symbol := strings.ToUpper(fields[0][:1]) + strings.ToLower(fields[0][1:])
		atoms[symbol]++
	}
	names := make([]string, 0, len(atoms))
	for sym := range atoms {
		names = append(names, sym)
	}
	sort.Strings(names)
	var parts []string
	for _, sym := range names {
		parts = append(parts, fmt.Sprintf("%s%d", sym, atoms[sym]))
	}
	return strings.Join(parts, ""), nil
}

// resolveProfileSources joins each explicit role to a known job and an actual
// OUT file. Reused filenames are accepted across different jobs, but a file
// from the same job cannot masquerade as both reactant and TS.
func (a *App) resolveProfileSources(req ScientificProfileRequest) (ScientificProfileRequest, map[string]OrcaAnalysis, error) {
	roles := []string{"S0", "T1"}
	if req.Kind == "reaction" {
		roles = []string{"Reactant", "TS", "Product"}
	}
	if len(req.Jobs) > len(roles) || len(req.Sources) > len(roles) {
		return req, nil, fmt.Errorf("요청한 결과 파일 또는 Job 역할이 너무 많습니다")
	}
	roleSet := map[string]bool{}
	for _, role := range roles {
		roleSet[role] = true
	}
	for role := range req.Jobs {
		if !roleSet[role] {
			return req, nil, fmt.Errorf("알 수 없는 Job 역할: %s", role)
		}
	}
	resolved := req
	resolved.Sources = map[string]string{}
	sources := map[string]OrcaAnalysis{}
	seen := map[string]bool{}
	loaded := map[string]Visualization{}
	formulas := map[string]string{}
	for _, role := range roles {
		file := req.Sources[role]
		if !safeName(file) || !strings.EqualFold(filepathExt(file), ".out") {
			return req, nil, fmt.Errorf("%s: Job 결과의 안전한 OUT 파일 이름을 지정하세요", role)
		}
		id := req.Jobs[role]
		if id == "" {
			id = req.JobID
		}
		folder, err := a.profileFolder(id)
		if err != nil {
			return req, nil, fmt.Errorf("%s: %w", role, err)
		}
		sameFile := folder + "\x00" + strings.ToLower(file)
		if seen[sameFile] {
			return req, nil, fmt.Errorf("%s: 같은 Job의 같은 OUT을 중복 배정했습니다", role)
		}
		seen[sameFile] = true
		viz, ok := loaded[folder]
		if !ok {
			viz, err = loadVisualization(folder)
			if err != nil {
				return req, nil, fmt.Errorf("%s: 결과 폴더를 읽지 못했습니다: %w", role, err)
			}
			loaded[folder] = viz
		}
		found := false
		for _, src := range viz.Sources {
			if src.Name == file && strings.EqualFold(filepathExt(src.Name), ".out") {
				// The opaque source key carries only a registered job ID and a filename.
				label := id
				if label == "" {
					label = "single"
				}
				key := label + "/" + file
				resolved.Sources[role] = key
				sources[key] = src.Analysis
				// Verify composition against actual local XYZ when available. If an XYZ
				// is unavailable, the request still requires explicit user confirmation.
				geo := src.Analysis.Geometry
				if safeName(geo) && strings.EqualFold(filepath.Ext(geo), ".xyz") {
					data, readErr := readPlotFile(filepath.Join(folder, geo), 4<<20)
					if readErr == nil {
						formula, formulaErr := xyzFormula(string(data))
						if formulaErr != nil {
							return req, nil, fmt.Errorf("%s: XYZ 조성을 확인할 수 없습니다: %w", role, formulaErr)
						}
						formulas[role] = formula
					}
				}
				found = true
				break
			}
		}
		if !found {
			return req, nil, fmt.Errorf("%s: 선택 Job에서 OUT %q을 찾지 못했습니다", role, file)
		}
	}
	// Formula equality is a necessary (not sufficient) guard for the 1:1
	// structure profiles supported here; IRC and state identity still need review.
	var expected string
	for _, role := range roles {
		formula := formulas[role]
		if formula == "" {
			return req, nil, fmt.Errorf("%s: 실제 XYZ 조성을 확인하지 못해 Profile 계산을 중단합니다", role)
		}
		if expected == "" {
			expected = formula
		} else if formula != expected {
			return req, nil, fmt.Errorf("구조의 원소 조성이 서로 다릅니다 (%s: %s, 기대 %s)", role, formula, expected)
		}
	}
	return resolved, sources, nil
}
func (a *App) scientificProfileRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 405, "POST만 지원합니다")
		return
	}
	var req ScientificProfileRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req) != nil {
		fail(w, 400, "Profile 요청 JSON을 읽지 못했습니다")
		return
	}
	resolved, sources, err := a.resolveProfileSources(req)
	if err != nil {
		fail(w, 422, err.Error())
		return
	}
	result, err := scientificProfile(resolved, sources)
	if err != nil {
		fail(w, 422, err.Error())
		return
	}
	respond(w, 200, result)
}
func filepathExt(s string) string {
	idx := strings.LastIndex(s, ".")
	if idx < 0 {
		return ""
	}
	return s[idx:]
}

// Compare non-keyword input settings conservatively, including solvent and
// SCF blocks. Different resource/optimization blocks also require review;
// rejecting an otherwise comparable pair is safer than hiding a method change.
func profileBlocks(settings string) string {
	var lines []string
	for _, line := range strings.Split(settings, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "!") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
