package main

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var hatEndFile = regexp.MustCompile(`(?im)^\s*NEB_END_XYZFILE\s+"([^"\r\n]+)"\s*$`)
var hatTemp = regexp.MustCompile(`(?im)^[\t ]*Temp[\t ]+([0-9.]+)[\t ]*(?:#.*)?\r?$`)
var hatDirection = regexp.MustCompile(`(?im)^[\t ]*Direction[\t ]+(\w+)[\t ]*(?:#.*)?\r?$`)
var hatStageBlock = regexp.MustCompile(`(?m)^%(pal|maxcore|output|freq|neb|geom|irc)(?: |$)`)

func hatCommonBlocks(inp string) string {
	lines := strings.Split(comparisonSettings(inp), "\n")
	var keep []string
	stageBlock := false
	for _, line := range lines {
		if strings.HasPrefix(line, "!") {
			stageBlock = false
			continue
		}
		if strings.HasPrefix(line, "%") {
			stageBlock = hatStageBlock.MatchString(line)
		}
		if !stageBlock {
			keep = append(keep, line)
		}
	}
	return strings.Join(keep, "\n")
}

// Every stage is one stationary-state workflow, not a separate batch molecule.
func validateHATRequest(req RunRequest, files map[string]string) error {
	present := req.HATMapping != nil || req.HATRestart != nil
	for _, s := range req.Stages {
		if s.HAT != "" || s.EndSource != "" {
			present = true
		}
	}
	if !present {
		return nil
	}
	if len(req.Stages) != 4 {
		return fmt.Errorf("HAT: R/P/TS/IRC 네 단계가 필요합니다")
	}
	roles := []string{"reactant", "product", "ts", "irc"}
	var level []string
	var blocks string
	var temperature float64
	var q, m int
	for i, s := range req.Stages {
		if s.HAT != roles[i] || s.Opt != (i < 3) || s.Goat || s.Stationary != "" || s.CompositeFrom != "" || (i != 2 && s.EndSource != "") {
			return fmt.Errorf("HAT: 단계 역할 또는 산출물 선언 오류")
		}
		inp := files[strings.ToLower(s.Input)]
		header := xyzHeader.FindAllStringSubmatch(inp, -1)
		if len(header) != 1 || !strings.EqualFold(strings.Trim(header[0][3], `"'`), s.Source) {
			return fmt.Errorf("HAT: 각 INP의 * xyzfile이 단계 XYZ와 일치해야 합니다")
		}
		cq, cm, e := chargeMultiplicity(inp)
		if e != nil {
			return fmt.Errorf("HAT: 전하/Multiplicity 누락")
		}
		if i == 0 {
			q, m = cq, cm
		} else if q != cq || m != cm {
			return fmt.Errorf("HAT: 전하/Multiplicity 불일치")
		}
		k := keywordLine(inp)
		if k["opt"] != (i < 2) || k["neb-ts"] != (i == 2) || k["irc"] != (i == 3) || k["freq"] != (i < 3) || k["optts"] || k["numfreq"] {
			return fmt.Errorf("HAT: 단계별 Opt/Freq/NEB-TS/IRC 키워드 불일치")
		}
		common := []string{}
		for name := range k {
			switch name {
			case "opt", "freq", "neb-ts", "irc":
			default:
				common = append(common, name)
			}
		}
		slices.Sort(common)
		if i == 3 {
			direction := hatDirection.FindAllStringSubmatch(inp, -1)
			if len(direction) != 1 || !strings.EqualFold(direction[0][1], "both") {
				return fmt.Errorf("HAT: IRC Direction both가 필요합니다")
			}
		}
		modelBlocks := hatCommonBlocks(inp)
		if i == 0 {
			level = common
			blocks = modelBlocks
		} else if !slices.Equal(common, level) || blocks != modelBlocks {
			return fmt.Errorf("HAT: 계산 수준 또는 용매 불일치")
		}
		if i < 3 {
			t := hatTemp.FindAllStringSubmatch(inp, -1)
			if len(t) != 1 {
				return fmt.Errorf("HAT: Frequency 온도를 명시하세요")
			}
			v, err := strconv.ParseFloat(t[0][1], 64)
			if err != nil || v <= 0 {
				return fmt.Errorf("HAT: Temperature가 유효하지 않습니다")
			}
			if i == 0 {
				temperature = v
			} else if v != temperature {
				return fmt.Errorf("HAT: Frequency 온도 불일치")
			}
		}
	}
	r, p, t, irc := req.Stages[0], req.Stages[1], req.Stages[2], req.Stages[3]
	if strings.EqualFold(r.Source, p.Source) {
		return fmt.Errorf("HAT: 서로 다른 두 XYZ가 필요합니다")
	}
	a, b := files[strings.ToLower(r.Source)], files[strings.ToLower(p.Source)]
	if req.HATMapping == nil {
		return fmt.Errorf("HAT: 이동 H, donor, acceptor의 인덱스를 지정하고 원자 대응을 확인하세요")
	}
	rm, pm, err := hatEndpoints(a, b)
	if err != nil {
		return err
	}
	if err := req.HATMapping.checkIndices(rm); err != nil {
		return err
	}
	if changes := hatChanges(rm, pm, -1, -1); !req.HATMapping.matches(changes) {
		return fmt.Errorf("HAT: 단일 H 이동과 다른 결합 변화가 있습니다 (%v). 원자 대응 검사에서 Product XYZ를 확인하세요", changes)
	}
	if !strings.EqualFold(t.Source, stageGeometry(r)) || !strings.EqualFold(t.EndSource, stageGeometry(p)) || !strings.EqualFold(irc.Source, stageGeometry(t)) {
		return fmt.Errorf("HAT: 최적화 XYZ 연결 오류")
	}
	ref := hatEndFile.FindStringSubmatch(files[strings.ToLower(t.Input)])
	if ref == nil || !safeName(ref[1]) || !strings.EqualFold(ref[1], t.EndSource) {
		return fmt.Errorf("HAT: NEB_END_XYZFILE 연결 오류")
	}
	return nil
}

type hatStageResult struct {
	Role             string
	GibbsEh          float64
	DonorDistance    float64  // Å
	AcceptorDistance float64  // Å
	S2               *float64 `json:",omitempty"`
	HMullikenSpin    *float64 `json:",omitempty"`
}

type hatResult struct {
	Stages          []hatStageResult
	TemperatureK    float64
	TSImaginaryCm1  float64
	ReactionKcalMol float64
	ForwardKcalMol  float64
	ReverseKcalMol  float64
	IRC             string
	Unavailable     string `json:",omitempty"`
}

func analyzeHAT(folder string, req RunRequest, completed []string, analyses map[string]OrcaAnalysis) *hatResult {
	if len(req.Stages) != 4 || req.Stages[0].HAT != "reactant" {
		return nil
	}
	result := &hatResult{}
	if req.HATMapping == nil {
		result.Unavailable = "원자 대응 정보 없음"
		return result
	}
	mapping := req.HATMapping
	for _, stage := range req.Stages[:3] {
		analysis := analyses[stage.Input]
		if !slices.Contains(completed, stage.Input) || analysis.Gibbs == nil || math.IsNaN(*analysis.Gibbs) || math.IsInf(*analysis.Gibbs, 0) {
			result.Unavailable = stage.Name + " 단계 미완료 또는 Gibbs 없음"
			return result
		}
		if analysis.Temperature == nil || *analysis.Temperature <= 0 || (len(result.Stages) > 0 && result.TemperatureK != *analysis.Temperature) {
			result.Unavailable = "Opt/Freq와 TS OUT에서 동일한 온도가 확인되어야 합니다"
			return result
		}
		result.TemperatureK = *analysis.Temperature
		raw, err := readGloriusFile(folder, stageGeometry(stage))
		if err != nil {
			result.Unavailable = err.Error()
			return result
		}
		_, coords, err := xyzGeometry(raw)
		if err != nil || max(mapping.H, mapping.Donor, mapping.Acceptor) >= len(coords) || min(mapping.H, mapping.Donor, mapping.Acceptor) < 0 {
			result.Unavailable = stage.Name + " XYZ 원자 대응을 확인할 수 없습니다"
			return result
		}
		h := tripVec(coords[mapping.H])
		row := hatStageResult{Role: stage.HAT, GibbsEh: *analysis.Gibbs,
			DonorDistance:    tripDistance(h, tripVec(coords[mapping.Donor])),
			AcceptorDistance: tripDistance(h, tripVec(coords[mapping.Acceptor])), S2: analysis.S2}
		for _, atom := range analysis.Populations["Mulliken"] {
			if atom.Index == mapping.H {
				row.HMullikenSpin = atom.Spin
				break
			}
		}
		if stage.HAT == "ts" {
			for _, frequency := range analysis.Frequencies {
				if frequency < -0.1 {
					result.TSImaginaryCm1 = frequency
					break
				}
			}
		}
		result.Stages = append(result.Stages, row)
	}
	r, p, t := result.Stages[0].GibbsEh, result.Stages[1].GibbsEh, result.Stages[2].GibbsEh
	result.ReactionKcalMol = (p - r) * 627.509474
	result.ForwardKcalMol = (t - r) * 627.509474
	result.ReverseKcalMol = (t - p) * 627.509474
	result.IRC = "IRC 완료 전 · 연결성 미검증"
	if slices.Contains(completed, req.Stages[3].Input) {
		result.IRC = hatIRCEndpoints(folder, req)
	}
	return result
}

func formatHATResult(result *hatResult) string {
	if result == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n[HAT / NEB-TS Gibbs barrier]\n")
	for _, s := range result.Stages {
		fmt.Fprintf(&b, "%s G = %.10f Eh | H–Donor %.3f Å | H–Acceptor %.3f Å\n", s.Role, s.GibbsEh, s.DonorDistance, s.AcceptorDistance)
		if s.S2 != nil {
			fmt.Fprintf(&b, "  <S²> %.5f\n", *s.S2)
		}
		if s.HMullikenSpin != nil {
			fmt.Fprintf(&b, "  이동 H Mulliken Spin %.5f\n", *s.HMullikenSpin)
		}
	}
	if result.Unavailable != "" {
		fmt.Fprintf(&b, "Unavailable: %s\n", result.Unavailable)
		return b.String()
	}
	fmt.Fprintf(&b, "Temperature %.2f K | TS imaginary frequency %.2f cm⁻¹\n", result.TemperatureK, result.TSImaginaryCm1)
	fmt.Fprintf(&b, "Reaction ΔG = %.4f kcal/mol\nForward ΔG‡ = %.4f kcal/mol\nReverse ΔG‡ = %.4f kcal/mol\n", result.ReactionKcalMol, result.ForwardKcalMol, result.ReverseKcalMol)
	if result.ForwardKcalMol < 0 || result.ReverseKcalMol < 0 {
		b.WriteString("Warning: TS free energy below an endpoint; inspect thermal corrections and connectivity.\n")
	}
	fmt.Fprintf(&b, "IRC: %s\n", result.IRC)
	b.WriteString("Barriers use converged frequency Gibbs energies. Review the imaginary mode, IRC endpoints, and spin; NEB HEI energies are path-search diagnostics.\n")
	return b.String()
}
