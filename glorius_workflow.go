package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// The SI specifies the calculation levels, not the exact original INPs.
// GOAT produces a different filename from a DFT optimization (ORCA 6 manual).
func stageGeometry(s Stage) string {
	stem := strings.TrimSuffix(s.Input, filepath.Ext(s.Input))
	if s.Goat {
		return stem + ".globalminimum.xyz"
	}
	if s.HAT == "ts" {
		return stem + "_NEB-TS_converged.xyz"
	}
	if s.Opt {
		return stem + ".xyz"
	}
	return ""
}
func keywordLine(inp string) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(inp, "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if strings.HasPrefix(line, "!") {
			for _, v := range strings.Fields(line[1:]) {
				out[strings.ToLower(v)] = true
			}
		}
	}
	return out
}
func gloriusLevel(inp string, high bool) bool {
	k := keywordLine(normalizeORCAInput(inp))
	basis := "def2-tzvp"
	if high {
		basis = "def2-qzvppd"
	}
	required := []string{"m062x", basis, "def2/j", "rijcosx", "d3zero", "cpcm(acetonitrile)"}
	allowed := map[string]bool{}
	for _, v := range required {
		if !k[v] {
			return false
		}
		allowed[v] = true
	}
	for _, v := range []string{"uks", "rks", "tightscf", "normalscf", "verytightscf", "opt", "optts", "freq", "irc"} {
		allowed[v] = true
	}
	for v := range k {
		if !allowed[v] {
			return false
		}
	}
	// Additional calculations cannot silently change the fixed SI level.
	if k["uks"] && k["rks"] {
		return false
	}
	return true
}

// Validate explicit metadata, never infer a successful stage from its UI name.
func validateGloriusDeclaration(s Stage, inp string, earlier map[string]Stage, files map[string]string) error {
	k := keywordLine(inp)
	if s.Goat || s.Stationary != "" || s.CompositeFrom != "" {
		header := xyzHeader.FindStringSubmatch(inp)
		if header == nil || !strings.EqualFold(strings.Trim(header[3], `"'`), s.Source) {
			return fmt.Errorf("검증 단계의 * xyzfile은 지정된 입력 XYZ와 일치해야 합니다")
		}
	}
	if s.Goat {
		if s.Opt || !k["goat"] || !k["xtb2"] || s.Stationary != "" || s.CompositeFrom != "" {
			return fmt.Errorf("GOAT 단계는 GOAT XTB2 검색이어야 합니다")
		}
	} else if k["goat"] {
		return fmt.Errorf("GOAT 산출물 검증이 없는 입력입니다. GOAT 전용 프리셋을 사용하세요")
	}
	if s.Stationary != "" {
		if !s.Opt || !k["freq"] || !gloriusLevel(inp, false) || k["irc"] {
			return fmt.Errorf("검증된 stationary 단계에는 SI 수준 Opt/Freq가 필요합니다")
		}
		if s.Stationary == "minimum" && (!k["opt"] || k["optts"]) {
			return fmt.Errorf("Minimum은 Opt/Freq가 필요합니다")
		}
		if s.Stationary == "ts" && (!k["optts"] || k["opt"]) {
			return fmt.Errorf("TS는 OptTS/Freq가 필요합니다")
		}
		if s.Stationary != "minimum" && s.Stationary != "ts" {
			return fmt.Errorf("알 수 없는 stationary 종류")
		}
	}
	if s.CompositeFrom != "" {
		low, ok := earlier[strings.ToLower(s.CompositeFrom)]
		if !ok || low.Stationary == "" || !low.Opt || !strings.EqualFold(stageGeometry(low), s.Source) || s.Goat || s.Opt || s.Stationary != "" || !gloriusLevel(inp, true) {
			return fmt.Errorf("Composite SP는 검증된 SI Opt/Freq의 산출 XYZ를 사용해야 합니다")
		}
		if k["opt"] || k["optts"] || k["freq"] || k["numfreq"] || k["irc"] {
			return fmt.Errorf("Composite high-level 단계는 SP만 허용합니다")
		}
		q1, m1, e1 := chargeMultiplicity(files[strings.ToLower(low.Input)])
		q2, m2, e2 := chargeMultiplicity(inp)
		if e1 != nil || e2 != nil || q1 != q2 || m1 != m2 {
			return fmt.Errorf("Composite Opt/SP의 Charge/Multiplicity가 다릅니다")
		}
	}
	return nil
}
func validateGOAT(folder string, s Stage) error {
	stem := strings.TrimSuffix(s.Input, filepath.Ext(s.Input))
	out, err := os.Open(filepath.Join(folder, stem+".out"))
	if err != nil {
		return err
	}
	defer out.Close()
	// GOAT's ensemble search does not require a DFT SCF/optimization marker.
	normal, found, ensemble := false, false, false
	scanner := bufio.NewScanner(out)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	for scanner.Scan() {
		line := strings.ToUpper(scanner.Text())
		normal = normal || strings.Contains(line, "ORCA TERMINATED NORMALLY")
		found = found || strings.Contains(line, "GLOBAL MINIMUM FOUND!")
		ensemble = ensemble || strings.Contains(line, "FINAL ENSEMBLE INFO")
	}
	if err = scanner.Err(); err != nil {
		return err
	}
	if !normal || !found || !ensemble {
		return fmt.Errorf("GOAT 정상 종료·global minimum·ensemble 완료를 확인하지 못했습니다")
	}
	source, err := os.ReadFile(filepath.Join(folder, s.Source))
	if err != nil {
		return err
	}
	result, err := os.ReadFile(filepath.Join(folder, stageGeometry(s)))
	if err != nil {
		return fmt.Errorf("GOAT globalminimum XYZ 누락: %w", err)
	}
	n, z, err := parseXYZAtoms(string(result))
	if err != nil {
		return err
	}
	sn, sz, err := parseXYZAtoms(string(source))
	if err != nil || n != sn || z != sz || !sameElements(source, result) {
		return fmt.Errorf("GOAT XYZ 원자 수/원소 순서 불일치")
	}
	return nil
}

type gloriusCompositeRow struct {
	Stage         string
	Source        string
	Stationary    string
	InitialSource string
	Result        *SICompositeResult
	Error         string
}

func readGloriusFile(folder, name string) (string, error) {
	if !safeName(name) {
		return "", fmt.Errorf("unsafe file name")
	}
	f, err := os.Open(filepath.Join(folder, name))
	if err != nil {
		return "", err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, (16<<20)+1))
	if err != nil {
		return "", err
	}
	if len(b) > 16<<20 {
		return "", fmt.Errorf("%s exceeds 16 MiB composite parser limit", name)
	}
	return string(b), nil
}

// Only user-declared SP pairs with matched charge, geometry, and valid
// stationary-point output are combined. Never use an unrun/missing stage.
func gloriusComposites(folder string, req RunRequest, completed []string) []gloriusCompositeRow {
	rows := []gloriusCompositeRow{}
	earlier := map[string]Stage{}
	for _, s := range req.Stages {
		if s.CompositeFrom != "" {
			low, ok := earlier[strings.ToLower(s.CompositeFrom)]
			row := gloriusCompositeRow{Stage: s.Name, Source: s.Input, Stationary: low.Stationary, InitialSource: low.Source}
			if !ok || !strings.EqualFold(stageGeometry(low), s.Source) {
				row.Error = "invalid stage mapping"
				rows = append(rows, row)
				earlier[strings.ToLower(s.Input)] = s
				continue
			}
			if !slices.Contains(completed, low.Input) || !slices.Contains(completed, s.Input) {
				row.Error = "Opt/Freq 또는 SP가 실행 검증을 통과하지 않았습니다"
				rows = append(rows, row)
				earlier[strings.ToLower(s.Input)] = s
				continue
			}
			lowStem := strings.TrimSuffix(low.Input, filepath.Ext(low.Input))
			highStem := strings.TrimSuffix(s.Input, filepath.Ext(s.Input))
			names := []string{low.Input, lowStem + ".out", stageGeometry(low), s.Input, highStem + ".out", s.Source}
			var values [6]string
			for i, name := range names {
				v, e := readGloriusFile(folder, name)
				if e != nil {
					row.Error = e.Error()
					break
				}
				values[i] = v
			}
			if row.Error == "" {
				inputs := map[string]string{strings.ToLower(low.Input): values[0]}
				if e := validateGloriusDeclaration(low, values[0], nil, nil); e != nil {
					row.Error = e.Error()
				} else if e := validateGloriusDeclaration(s, values[3], earlier, inputs); e != nil {
					row.Error = e.Error()
				} else {
					result, e := calculateSIComposite(SICompositeRequest{LowINP: values[0], LowOUT: values[1], LowXYZ: values[2], HighINP: values[3], HighOUT: values[4], HighXYZ: values[5], StationaryPoint: low.Stationary})
					if e != nil {
						row.Error = e.Error()
					} else {
						row.Result = &result
					}
				}
			}
			rows = append(rows, row)
		}
		earlier[strings.ToLower(s.Input)] = s
	}
	return rows
}
func gloriusCompositeReport(rows []gloriusCompositeRow) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n[Glorius 2026 SI Composite Gibbs — Eh]\nG = E_high + (G_low - E_low)\n")
	for _, r := range rows {
		if r.Result == nil {
			fmt.Fprintf(&b, "%s: Unavailable (%s)\n", r.Stage, r.Error)
			continue
		}
		a := r.Result
		fmt.Fprintf(&b, "%s | charge %d mult %d | E_high %.10f | G_corr %.10f | G_composite %.10f\n", r.Stage, a.Charge, a.Multiplicity, a.HighElectronicEh, a.CorrectionEh, a.CompositeGibbsEh)
		for _, w := range a.Warnings {
			fmt.Fprintf(&b, "  %s\n", w)
		}
	}
	// The dedicated paired workflow always uses the same starting conformer.
	// Additional guards below require equal charge/atom count, singlet/triplet.
	// Do not treat this as an independently verified triplet-state assignment.
	if len(rows) == 2 && rows[0].Result != nil && rows[1].Result != nil && rows[0].Stationary == "minimum" && rows[1].Stationary == "minimum" {
		a, c := rows[0].Result, rows[1].Result
		if a.Multiplicity == 1 && c.Multiplicity == 3 && a.Charge == c.Charge && a.Atoms == c.Atoms && rows[0].InitialSource != "" && strings.EqualFold(rows[0].InitialSource, rows[1].InitialSource) {
			d := c.CompositeGibbsEh - a.CompositeGibbsEh
			if !math.IsNaN(d) && !math.IsInf(d, 0) {
				fmt.Fprintf(&b, "Triplet - Singlet composite ΔG (same starting species must be confirmed): %.10f Eh / %.4f kcal/mol\n", d, d*627.509474)
			}
		}
	}
	b.WriteString("SI-level reconstruction; original input keywords, conformer completeness, state assignment and standard-state corrections require review. IRC endpoints require visual verification.\n")
	return b.String()
}
