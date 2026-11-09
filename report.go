package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Parsed values are measurements reported by ORCA. Missing fields are never
// filled with guessed numbers, and energies from different geometries/methods
// are not silently treated as interchangeable.
const ehToEV = 27.211386245988
const ehToKcal = 627.5094740631
const ehToKJ = 2625.4996394799

var orcaNumber = `[-+]?(?:\d+(?:\.\d*)?|\.\d+)(?:[EeDd][-+]?\d+)?`
var (
	hirshfeldAtomRx    = regexp.MustCompile(`^\s*(\d+)\s+([A-Z][a-z]?)\s+(` + orcaNumber + `)\s+(` + orcaNumber + `)\s*$`)
	modernAbsorptionRx = regexp.MustCompile(`^\s*0-\S+\s+->\s+(\d+)-\S+\s+(` + orcaNumber + `)\s+(` + orcaNumber + `)\s+(` + orcaNumber + `)\s+(` + orcaNumber + `)\s+`)

	finalEnergyRx  = regexp.MustCompile(`(?i)FINAL SINGLE POINT ENERGY\s+(` + orcaNumber + `)`)
	enthalpyRx     = regexp.MustCompile(`(?i)^\s*Total Enthalpy\s+\.{0,3}\s*(` + orcaNumber + `)\s+Eh\b`)
	gibbsRx        = regexp.MustCompile(`(?i)^\s*Final Gibbs free energy\s+\.{0,3}\s*(` + orcaNumber + `)\s+Eh\b`)
	zpeRx          = regexp.MustCompile(`(?i)^\s*Zero point energy\s+\.{0,3}\s*(` + orcaNumber + `)\s+Eh\b`)
	thermalRx      = regexp.MustCompile(`(?i)^\s*Thermal correction to (?:Energy|Internal Energy)\s*\.{0,3}\s*(` + orcaNumber + `)\s+Eh\b`)
	entropyValueRx = regexp.MustCompile(`(?i)^\s*Total entropy\s*\.{0,3}\s*(` + orcaNumber + `)\s+Eh/K\b`)
	entropyRx      = regexp.MustCompile(`(?i)^\s*Final entropy term\s+\.{0,3}\s*(` + orcaNumber + `)\s+Eh\b`)
	tempRx         = regexp.MustCompile(`(?i)Temperature\s*(?:[:=]|\.{1,3})?\s*(` + orcaNumber + `)\s*K\b`)
	s2Rx           = regexp.MustCompile(`(?i)(?:Expectation value of\s*)?<S(?:\*\*|\^)?2>\s*(?:\:|=)\s*(` + orcaNumber + `)`)
	rootRx         = regexp.MustCompile(`(?i)^\s*STATE\s+(\d+)\s*:\s*E\s*=\s*(` + orcaNumber + `)\s*(?:au|a\.u\.)\s+(` + orcaNumber + `)\s*eV`)
	multRx         = regexp.MustCompile(`(?i)\bMult\s+([12345])\b`)
	atomRx         = regexp.MustCompile(`^\s*(\d+)\s+([A-Z][a-z]?)\s*:\s*(` + orcaNumber + `)(?:\s+(` + orcaNumber + `))?\s*$`)
	freqRx         = regexp.MustCompile(`^\s*(\d+)\s*:\s*(` + orcaNumber + `)\s*cm\*\*-1`)
	orbitalRx      = regexp.MustCompile(`^\s*(\d+)\s+(` + orcaNumber + `)\s+(` + orcaNumber + `)\s+(` + orcaNumber + `)\s*$`)
	mayerRx        = regexp.MustCompile(`(?i)B\(\s*(\d+)\s*[-:]\s*([A-Z][a-z]?)\s*,\s*(\d+)\s*[-:]\s*([A-Z][a-z]?)\s*\)\s*:\s*(` + orcaNumber + `)`)
	chargeMultRx   = regexp.MustCompile(`(?im)^\s*\*\s*xyz(?:file)?\s+(-?\d+)\s+(\d+)\b`)
)

// Atom numbers in ORCA output start at zero. They are NOT uracil ring-position
// labels, so e.g. "5 C" must never automatically be called "C5".
type AtomicPopulation struct {
	Index   int
	Element string
	Charge  float64
	Spin    *float64
}
type ExcitedRoot struct {
	Root               int
	Kind               string
	EV                 float64
	EH                 float64
	OscillatorStrength *float64 `json:",omitempty"`
	WavelengthNM       *float64 `json:",omitempty"`
}
type BondOrder struct {
	I     int
	J     int
	AtomI string
	AtomJ string
	Value float64
}
type FrontierOrbital struct {
	HOMO   float64
	LUMO   float64
	FoundH bool
	FoundL bool
}
type OrcaAnalysis struct {
	Filename                  string
	Found                     bool
	Normal                    bool
	Optimized                 bool
	SCFConverged              bool
	Electronic                *float64
	Enthalpy                  *float64
	Gibbs                     *float64
	ZPE                       *float64
	EntropyTerm               *float64
	ThermalCorrection         *float64
	ThermalEnthalpyCorrection *float64
	Entropy                   *float64 // Eh/K, distinct from the T*S energy term
	Temperature               *float64
	S2                        *float64
	Multiplicity              int
	Charge                    int
	Method                    string
	Geometry                  string
	Settings                  string
	ImaginaryCount            *int
	FinalXYZ                  string `json:",omitempty"`
	Status                    string
	Availability              map[string]string
	Roots                     []ExcitedRoot
	Populations               map[string][]AtomicPopulation
	Frequencies               []float64
	Mayer                     []BondOrder
	Frontier                  FrontierOrbital
	Warnings                  []string
}

func number(s string) (float64, error) {
	return strconv.ParseFloat(strings.ReplaceAll(strings.ReplaceAll(s, "D", "E"), "d", "e"), 64)
}
func matchedNumber(rx *regexp.Regexp, s string) *float64 {
	m := rx.FindStringSubmatch(s)
	if len(m) < 2 {
		return nil
	}
	n, e := number(m[1])
	if e != nil {
		return nil
	}
	return &n
}
func lastFloat(lines []string, rx *regexp.Regexp) *float64 {
	var last *float64
	for _, s := range lines {
		if p := matchedNumber(rx, s); p != nil {
			last = p
		}
	}
	return last
}

// ParseOrcaOutput is intentionally text-only and independent from external
// chemistry libraries; it handles large output as a stream.
func ParseOrcaOutput(input io.Reader, filename string, inp string) (OrcaAnalysis, error) {
	a := OrcaAnalysis{Filename: filename, Populations: map[string][]AtomicPopulation{}, Status: "Failed"}
	a.Settings = comparisonSettings(inp)
	if m := regexp.MustCompile(`(?im)^\s*\*\s*xyzfile\s+-?\d+\s+\d+\s+(\S+)`).FindStringSubmatch(inp); len(m) > 1 {
		a.Geometry = m[1]
	}
	if m := chargeMultRx.FindStringSubmatch(inp); len(m) > 2 {
		a.Charge, _ = strconv.Atoi(m[1])
		a.Multiplicity, _ = strconv.Atoi(m[2])
	}
	for _, line := range strings.Split(inp, "\n") {
		s := strings.TrimSpace(line)
		if strings.HasPrefix(s, "!") && a.Method == "" {
			a.Method = strings.TrimSpace(strings.TrimPrefix(s, "!"))
		}
	}
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 4*1024*1024)
	var thermo []string
	section := ""
	orbitalSection := false
	seenFreq := false
	seenOrb := false
	rootKind := "unknown"
	absorption := false
	var absorptionRows = map[int][2]float64{}
	atomRows := map[string][]AtomicPopulation{}
	var orbRows [][2]float64
	for scanner.Scan() {
		line := scanner.Text()
		u := strings.ToUpper(strings.TrimSpace(line))
		if strings.Contains(u, "ORCA TERMINATED NORMALLY") {
			a.Normal = true
		}
		if strings.Contains(u, "ERROR TERMINATION") {
			a.Normal = false
		}
		if strings.Contains(u, "THE OPTIMIZATION HAS CONVERGED") {
			a.Optimized = true
		}
		if strings.Contains(u, "SCF CONVERGED") {
			a.SCFConverged = true
		}
		if strings.Contains(u, "SCF NOT CONVERGED") || strings.Contains(u, "SCF DID NOT CONVERGE") {
			a.SCFConverged = false
		}
		if (strings.Contains(u, "ROOT FOLLOW") && (strings.Contains(u, "FAIL") || strings.Contains(u, "LOST"))) || strings.Contains(u, "STATE SWITCH") {
			a.Warnings = append(a.Warnings, "Root Following / State Switching 메시지가 있습니다. 의도한 Excited State를 원본 OUT에서 확인하세요.")
		}
		if strings.Contains(u, "ABSORPTION SPECTRUM VIA TRANSITION ELECTRIC DIPOLE MOMENTS") {
			absorption = true
			absorptionRows = map[int][2]float64{}
			continue
		}
		if absorption && strings.Contains(u, "SPECTRUM") {
			absorption = false
		}
		if absorption {
			if m := modernAbsorptionRx.FindStringSubmatch(line); len(m) == 6 && rootKind == "S" {
				root, _ := strconv.Atoi(m[1])
				wave, e1 := number(m[4])
				strength, e2 := number(m[5])
				if e1 == nil && e2 == nil && wave > 0 {
					absorptionRows[root] = [2]float64{wave, strength}
				}
			}
			// ORCA's legacy table: State Energy(cm-1) Wavelength(nm) fosc ... .
			// Other layouts remain unavailable rather than guessing column units.
			fields := strings.Fields(line)
			if len(fields) == 8 {
				root, e := strconv.Atoi(fields[0])
				wavelength, e2 := number(fields[2])
				strength, e3 := number(fields[3])
				if e == nil && e2 == nil && e3 == nil && wavelength > 0 && rootKind == "S" {
					absorptionRows[root] = [2]float64{wavelength, strength}
				}
			}
		}
		if v := matchedNumber(finalEnergyRx, line); v != nil {
			a.Electronic = v
		}
		if v := matchedNumber(s2Rx, line); v != nil && !strings.HasPrefix(u, "STATE") {
			a.S2 = v
		}
		if len(thermo) < 200000 && (strings.Contains(u, "ENTHALPY") || strings.Contains(u, "GIBBS") || strings.Contains(u, "ZERO POINT") || strings.Contains(u, "ENTROPY") || strings.Contains(u, "TEMPERATURE") || strings.Contains(u, "THERMAL CORRECTION")) {
			thermo = append(thermo, line)
		}
		if strings.Contains(u, "TD-DFT") || strings.Contains(u, "TDA EXCITED STATES") || strings.Contains(u, "EXCITED STATES") {
			if strings.Contains(u, "TRIPLET") {
				rootKind = "T"
			} else if strings.Contains(u, "SINGLET") {
				rootKind = "S"
			}
		}
		if m := rootRx.FindStringSubmatch(line); len(m) >= 4 {
			root, _ := strconv.Atoi(m[1])
			en, _ := number(m[2])
			ev, _ := number(m[3])
			kind := rootKind
			if m2 := multRx.FindStringSubmatch(line); len(m2) > 1 {
				if m2[1] == "1" {
					kind = "S"
				}
				if m2[1] == "3" {
					kind = "T"
				}
			}
			a.Roots = append(a.Roots, ExcitedRoot{Root: root, Kind: kind, EV: ev, EH: en})
		}
		if strings.Contains(u, "MULLIKEN ATOMIC CHARGES") {
			section = "Mulliken"
			atomRows[section] = nil
			continue
		}
		if strings.Contains(u, "LOEWDIN ATOMIC CHARGES") || strings.Contains(u, "LÖWDIN ATOMIC CHARGES") {
			section = "Loewdin"
			atomRows[section] = nil
			continue
		}
		if strings.Contains(u, "HIRSHFELD ATOMIC CHARGES") || strings.Contains(u, "HIRSHFELD ANALYSIS") {
			section = "Hirshfeld"
			atomRows[section] = nil
			continue
		}
		if strings.Contains(u, "VIBRATIONAL FREQUENCIES") {
			seenFreq = true
			section = "frequency"
			a.Frequencies = nil
			continue
		}
		if strings.Contains(u, "ORBITAL ENERGIES") {
			orbitalSection = true
			seenOrb = true
			orbRows = nil
			section = "orbitals"
			continue
		}
		if strings.HasPrefix(u, "MAYER POPULATION ANALYSIS") {
			a.Mayer = nil
		}
		if strings.HasPrefix(u, "NORMAL MODES") || strings.HasPrefix(u, "THERMOCHEMISTRY") || strings.HasPrefix(u, "IR SPECTRUM") || strings.HasPrefix(u, "MAYER POPULATION ANALYSIS") {
			section = ""
			orbitalSection = false
		}
		if m := mayerRx.FindAllStringSubmatch(line, -1); len(m) > 0 {
			for _, g := range m {
				i, _ := strconv.Atoi(g[1])
				j, _ := strconv.Atoi(g[3])
				v, _ := number(g[5])
				a.Mayer = append(a.Mayer, BondOrder{I: i, J: j, AtomI: g[2], AtomJ: g[4], Value: v})
			}
		}
		if section == "frequency" {
			if m := freqRx.FindStringSubmatch(line); len(m) > 2 {
				if n, e := number(m[2]); e == nil {
					a.Frequencies = append(a.Frequencies, n)
				}
			}
		}
		if section == "orbitals" && orbitalSection {
			if m := orbitalRx.FindStringSubmatch(line); len(m) > 4 {
				occ, _ := number(m[2])
				energy, _ := number(m[4])
				orbRows = append(orbRows, [2]float64{occ, energy})
			}
		}
		if section == "Mulliken" || section == "Loewdin" || section == "Hirshfeld" {
			m := atomRx.FindStringSubmatch(line)
			if section == "Hirshfeld" && len(m) == 0 {
				m = hirshfeldAtomRx.FindStringSubmatch(line)
			}
			if len(m) > 3 {
				index, _ := strconv.Atoi(m[1])
				q, _ := number(m[3])
				var spin *float64
				if m[4] != "" {
					x, _ := number(m[4])
					spin = &x
				}
				atomRows[section] = append(atomRows[section], AtomicPopulation{Index: index, Element: m[2], Charge: q, Spin: spin})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return a, fmt.Errorf("OUT 파일 읽기 오류: %w", err)
	}
	a.Found = true
	a.Enthalpy = lastFloat(thermo, enthalpyRx)
	a.Gibbs = lastFloat(thermo, gibbsRx)
	a.ZPE = lastFloat(thermo, zpeRx)
	a.EntropyTerm = lastFloat(thermo, entropyRx)
	a.Temperature = lastFloat(thermo, tempRx)
	a.ThermalCorrection = lastFloat(thermo, thermalRx)
	a.Entropy = lastFloat(thermo, entropyValueRx)
	if a.Electronic != nil && a.Enthalpy != nil {
		v := *a.Enthalpy - *a.Electronic
		a.ThermalEnthalpyCorrection = &v
	}
	for method, rows := range atomRows {
		if len(rows) > 0 {
			a.Populations[method] = rows
		}
	}
	if seenOrb && len(orbRows) > 0 && a.Multiplicity <= 1 {
		// Do not report frontier levels for open-shell lists here: alpha/beta
		// orbital blocks need separate treatment and can be misleading.
		lastH := -1
		firstL := -1
		closedShell := true
		for i, p := range orbRows {
			if p[0] > 0.01 && p[0] < 1.99 {
				closedShell = false
			}
			if p[0] > 1.99 {
				lastH = i
			}
			if p[0] < 0.01 && firstL == -1 {
				firstL = i
			}
		}
		if closedShell && lastH >= 0 && firstL >= 0 && firstL > lastH {
			a.Frontier = FrontierOrbital{HOMO: orbRows[lastH][1], LUMO: orbRows[firstL][1], FoundH: true, FoundL: true}
		}
	}
	if len(a.Roots) > 0 {
		// De-duplicate repeated TD-DFT table snapshots by kind+root,
		// selecting the last occurrence (often the optimized geometry).
		seen := map[string]ExcitedRoot{}
		order := []string{}
		for _, r := range a.Roots {
			k := fmt.Sprintf("%s-%d", r.Kind, r.Root)
			if _, ok := seen[k]; !ok {
				order = append(order, k)
			}
			seen[k] = r
		}
		a.Roots = nil
		for _, k := range order {
			r := seen[k]
			// Absorption rows are assigned only to explicitly Singlet roots.
			if r.Kind == "S" {
				if v, ok := absorptionRows[r.Root]; ok {
					wavelength, strength := v[0], v[1]
					r.WavelengthNM = &wavelength
					r.OscillatorStrength = &strength
				}
			}
			a.Roots = append(a.Roots, r)
		}
	}
	if !a.Normal {
		a.Warnings = append(a.Warnings, "ORCA 정상 종료 문구가 없습니다. 값이 최종 결과가 아닐 수 있습니다.")
	}
	if seenFreq && len(a.Frequencies) == 0 {
		a.Warnings = append(a.Warnings, "Frequency 출력 섹션은 있으나 진동수를 추출하지 못했습니다.")
	}
	requiresOpt := regexp.MustCompile(`(?i)\b(?:Opt|TightOpt|LooseOpt|OptTS)\b`).MatchString(inp)
	if a.Normal && a.SCFConverged && a.Electronic != nil && (!requiresOpt || a.Optimized) {
		a.Status = "Completed"
	}
	if !a.SCFConverged {
		a.Warnings = append(a.Warnings, "SCF 수렴을 확인할 수 없습니다. OUT의 최종 계산 상태를 확인하세요.")
	}
	if len(a.Frequencies) > 0 {
		n := 0
		for _, v := range a.Frequencies {
			if v < -0.1 {
				n++
			}
		}
		a.ImaginaryCount = &n
	}
	if a.S2 != nil && a.Multiplicity > 0 {
		spin := float64(a.Multiplicity-1) / 2
		ideal := spin * (spin + 1)
		if math.Abs(*a.S2-ideal) > 0.1 {
			a.Warnings = append(a.Warnings, fmt.Sprintf("Spin Contamination 가능성: <S²>=%.4f, Multiplicity 기준 S(S+1)=%.4f. 0.1 초과 차이는 검토용 기준이며 판정은 별도 확인하세요.", *a.S2, ideal))
		}
	}
	if a.ImaginaryCount != nil {
		expected := 0
		if regexp.MustCompile(`(?i)\bOptTS\b`).MatchString(inp) {
			expected = 1
		}
		if *a.ImaginaryCount != expected {
			a.Warnings = append(a.Warnings, fmt.Sprintf("Frequency 검토: Imaginary Frequency %d개, %s 기대값 %d개. 진동 모드와 구조를 확인하세요.", *a.ImaginaryCount, map[bool]string{true: "Transition State", false: "Minimum"}[expected == 1], expected))
		}
	}
	a.Availability = analysisAvailability(a, inp)
	return a, nil
}

func showEh(v *float64) string {
	if v == nil {
		return "계산되지 않음 (OUT에 값 없음)"
	}
	return fmt.Sprintf("%.9f Eh", *v)
}
func showConversion(v *float64) string {
	if v == nil {
		return "계산되지 않음"
	}
	return fmt.Sprintf("%.9f Eh", *v)
}
func showDelta(eh float64) string {
	return fmt.Sprintf("%+.3f kcal/mol (%+.3f eV; %+.6f Eh; %+.3f kJ/mol)", eh*ehToKcal, eh*ehToEV, eh, eh*ehToKJ)
}

func formatAnalysis(a OrcaAnalysis, stage Stage) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ORCA Nova — 계산 결과 해석 (자동 추출)\n%s\n\n", strings.Repeat("=", 48))
	fmt.Fprintf(&b, "단계: %s\n파일: %s\nORCA 정상 종료 문구: %t\nGeometry Optimization 수렴: %t\n", stage.Name, a.Filename, a.Normal, a.Optimized)
	fmt.Fprintf(&b, "Status: %s\nGeometry source: %s\nSCF convergence: %t\n", a.Status, a.Geometry, a.SCFConverged)
	if a.FinalXYZ != "" {
		fmt.Fprintf(&b, "Final XYZ: %s\n", a.FinalXYZ)
	}
	if a.Method != "" {
		fmt.Fprintf(&b, "Method: %s\n", a.Method)
	}
	if a.Multiplicity > 0 {
		fmt.Fprintf(&b, "Charge: %d  Multiplicity: %d\n", a.Charge, a.Multiplicity)
	}
	b.WriteString("\n[Electronic Structure / Thermochemistry]\n")
	fmt.Fprintf(&b, "Electronic Energy (최종 SCF/상태 에너지): %s\n", showConversion(a.Electronic))
	fmt.Fprintf(&b, "Zero Point Energy (ZPE): %s\n", showEh(a.ZPE))
	fmt.Fprintf(&b, "Enthalpy (H): %s\n", showConversion(a.Enthalpy))
	fmt.Fprintf(&b, "Gibbs Free Energy (G): %s\n", showConversion(a.Gibbs))
	if a.Temperature != nil {
		fmt.Fprintf(&b, "Temperature: %.2f K\n", *a.Temperature)
	}
	if a.EntropyTerm != nil {
		fmt.Fprintf(&b, "Final entropy term (T·S): %s\n", showEh(a.EntropyTerm))
	}
	fmt.Fprintf(&b, "Thermal Correction to Internal Energy: %s\n", showEh(a.ThermalCorrection))
	fmt.Fprintf(&b, "Thermal Enthalpy Correction (H − Electronic Energy; includes ZPE): %s\n", showEh(a.ThermalEnthalpyCorrection))
	if a.Entropy != nil {
		fmt.Fprintf(&b, "Entropy (S): %.9f Eh/K = %.4f J/mol/K\n", *a.Entropy, *a.Entropy*ehToKJ*1000)
	} else {
		b.WriteString("Entropy (S): Unavailable (T·S와 다른 물리량)\n")
	}
	if a.S2 != nil {
		fmt.Fprintf(&b, "<S²> (Spin contamination 점검): %.6f\n", *a.S2)
	} else {
		b.WriteString("<S²>: 계산되지 않음 / 출력 없음\n")
	}
	b.WriteString("참고: Electronic/Gibbs/Enthalpy는 총 에너지(Eh), ZPE는 보정항(Eh)입니다. ΔE/ΔG 등 에너지 차이만 kcal/mol을 우선 표시합니다. 절대 Electronic Energy는 다른 분자의 반응 에너지로 바로 해석할 수 없습니다.\n")
	b.WriteString("\n[Excited States — TD-DFT Vertical Energy]\n")
	if len(a.Roots) == 0 {
		b.WriteString("출력된 S₁/T₁ Root가 없습니다.\n")
	} else {
		for _, r := range a.Roots {
			if r.Root > 30 {
				continue
			}
			label := fmt.Sprintf("Root %d", r.Root)
			if r.Kind != "unknown" {
				label = fmt.Sprintf("%s%d", r.Kind, r.Root)
			}
			fmt.Fprintf(&b, "%-12s %.3f kcal/mol (%.4f eV; %.6f Eh; %.3f kJ/mol) — Vertical excitation; NOT an Adiabatic gap\n", label, r.EV*ehToKcal/ehToEV, r.EV, r.EH, r.EH*ehToKJ)
			if r.WavelengthNM != nil {
				fmt.Fprintf(&b, "  Excitation Wavelength: %.3f nm\n", *r.WavelengthNM)
			} else {
				b.WriteString("  Excitation Wavelength: Unavailable\n")
			}
			if r.OscillatorStrength != nil {
				fmt.Fprintf(&b, "  Oscillator Strength: %.6f\n", *r.OscillatorStrength)
			} else {
				b.WriteString("  Oscillator Strength: Unavailable\n")
			}
		}
	}
	b.WriteString("\n[Atomic Partial Charges / Spin Population]\n")
	if len(a.Populations) == 0 {
		b.WriteString("원자별 Charge/Spin 표를 찾지 못했습니다.\n")
	} else {
		keys := make([]string, 0, len(a.Populations))
		for k := range a.Populations {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, method := range keys {
			fmt.Fprintf(&b, "%s 분석: (ORCA 원자 번호는 0부터 시작; C5 등 고리 위치와 다름)\n", method)
			rows := a.Populations[method]
			for _, row := range rows {
				if row.Spin != nil {
					fmt.Fprintf(&b, "  [%3d] %-2s  Charge=%+9.5f   Spin Population=%+9.5f\n", row.Index, row.Element, row.Charge, *row.Spin)
				} else {
					fmt.Fprintf(&b, "  [%3d] %-2s  Charge=%+9.5f   Spin=출력 없음\n", row.Index, row.Element, row.Charge)
				}
			}
		}
	}
	b.WriteString("Spin Density .cube는 공간 분포 파일이며, 위의 Atomic Spin Population과 동일한 수치가 아닙니다.\n")
	b.WriteString("\n[Frequency / Minimum / Transition State]\n")
	if len(a.Frequencies) == 0 {
		b.WriteString("Frequency 값이 없습니다. 허수 진동수 판정 불가.\n")
	} else {
		negative := []float64{}
		for _, v := range a.Frequencies {
			if v < -0.1 {
				negative = append(negative, v)
			}
		}
		fmt.Fprintf(&b, "Frequency 출력: %d개 / Imaginary Frequency: %d개\n", len(a.Frequencies), len(negative))
		if len(negative) > 0 {
			fmt.Fprintf(&b, "음의 진동수 (cm⁻¹): ")
			for i, v := range negative {
				if i > 0 {
					b.WriteString(", ")
				}
				fmt.Fprintf(&b, "%.2f", v)
			}
			b.WriteString("\n")
		}
		b.WriteString("작은 음의 진동수는 수치 오차 가능성이 있습니다. 구조·진동 모드를 확인하세요.\n")
	}
	b.WriteString("\n[Frontier Orbitals / Bond Orders]\n")
	if a.Frontier.FoundH && a.Frontier.FoundL {
		fmt.Fprintf(&b, "HOMO: %.4f eV (%.6f Eh) / LUMO: %.4f eV (%.6f Eh) / Gap: %.4f eV\n", a.Frontier.HOMO, a.Frontier.HOMO/ehToEV, a.Frontier.LUMO, a.Frontier.LUMO/ehToEV, a.Frontier.LUMO-a.Frontier.HOMO)
	} else {
		b.WriteString("HOMO/LUMO: 추출되지 않음 (Open-shell은 α/β 별도 분석 필요)\n")
	}
	if len(a.Mayer) > 0 {
		fmt.Fprintf(&b, "Mayer Bond Orders: %d개 추출\n", len(a.Mayer))
		for i, row := range a.Mayer {
			if i >= 40 {
				b.WriteString("... 나머지 생략\n")
				break
			}
			fmt.Fprintf(&b, "  %d%s–%d%s  %.4f\n", row.I, row.AtomI, row.J, row.AtomJ, row.Value)
		}
	} else {
		b.WriteString("Mayer Bond Orders: 출력 없음\n")
	}
	b.WriteString("\n[추가 계산 및 주의]\n")
	b.WriteString("Vertical T₁/S₁, Adiabatic T₁, Singlet–Triplet Gap은 동일한 값이 아닙니다.\n")
	b.WriteString("Reaction ΔG, Activation ΔG‡, BDE, SET, SOC는 필요한 별도 분자/상태/반응식의 결과를 모아야 판단 가능합니다.\n")
	b.WriteString("원자별 Spin Population은 반응 위치의 친핵성·선택성을 직접 증명하지 않습니다.\n")
	for _, w := range a.Warnings {
		fmt.Fprintf(&b, "주의: %s\n", w)
	}
	b.WriteString("\n[원본 확인]\n이 파일은 ORCA .out에서 추출한 숫자의 읽기 쉬운 요약입니다. 최종 논문 값으로 사용하기 전 계산 수렴·전자상태·Method를 검증하세요.\n")
	return b.String()
}

// writeAnalysisReports runs after every stage, including when a downstream
// stage fails. A partial output remains explicitly marked as such.
func writeAnalysisReports(folder string, req RunRequest) (string, error) {
	type result struct {
		stage    Stage
		analysis OrcaAnalysis
	}
	parsed := []result{}
	type stageResult struct {
		Stage    Stage
		Analysis OrcaAnalysis
	}
	all := []stageResult{}
	for _, stage := range req.Stages {
		stem := strings.TrimSuffix(stage.Input, filepath.Ext(stage.Input))
		outPath := filepath.Join(folder, stem+".out")
		f, err := os.Open(outPath)
		if err != nil {
			status := "Unavailable"
			if os.IsNotExist(err) {
				status = "Not calculated"
			}
			all = append(all, stageResult{stage, OrcaAnalysis{Filename: filepath.Base(outPath), Status: status}})
			continue
		}
		inp, _ := os.ReadFile(filepath.Join(folder, stage.Input))
		report, parseErr := ParseOrcaOutput(f, filepath.Base(outPath), string(inp))
		f.Close()
		if parseErr != nil {
			report.Warnings = append(report.Warnings, parseErr.Error())
		}
		if stage.Opt {
			xyz := stem + ".xyz"
			if info, e := os.Stat(filepath.Join(folder, xyz)); e == nil && !info.IsDir() {
				report.FinalXYZ = xyz
			}
		}
		all = append(all, stageResult{stage, report})
		if report.Found || parseErr != nil {
			if err := os.WriteFile(filepath.Join(folder, stem+"_analysis.txt"), []byte(formatAnalysis(report, stage)), 0644); err != nil {
				return "", err
			}
			parsed = append(parsed, result{stage, report})
		}
	}
	path := filepath.Join(folder, "ORCA_Nova_Analysis.txt")
	var b strings.Builder
	fmt.Fprintf(&b, "ORCA Nova | Workflow Calculation Report\nGenerated: %s\nLabel: %s\n\n", time.Now().Format(time.RFC3339), req.Label)
	if len(parsed) == 0 {
		b.WriteString("아직 읽을 수 있는 ORCA OUT 파일이 없습니다.\n")
	}
	for _, v := range all {
		if !v.Analysis.Found {
			fmt.Fprintf(&b, "단계: %s | Status: %s | OUT: %s\n", v.Stage.Name, v.Analysis.Status, v.Analysis.Filename)
		}
	}
	for _, v := range parsed {
		b.WriteString(formatAnalysis(v.analysis, v.stage))
		b.WriteString("\n\n")
	}
	b.WriteString("\n[Stage-to-stage electronic energy comparison]\n")
	// Only allow S0-triplet comparison for two separately converged stages at
	// matching charge, DFT level and basis. This remains an approximate
	// electronic-energy difference, never label as an adiabatic free energy.
	var s0, trip *OrcaAnalysis
	for i := range parsed {
		a := &parsed[i].analysis
		if !a.Normal || !a.SCFConverged || !a.Optimized || a.Electronic == nil {
			continue
		}
		if a.Multiplicity == 1 && s0 == nil {
			s0 = a
		}
		if a.Multiplicity == 3 && trip == nil {
			trip = a
		}
	}
	if s0 != nil && trip != nil {
		if s0.Method == "" || s0.Method != trip.Method || s0.Settings != trip.Settings || s0.Charge != trip.Charge {
			b.WriteString("S₀/T₁ Electronic Energy 차이: 자동 계산 제외 (Method / 추가 설정 / Charge 불일치 또는 정보 없음)\n")
		} else {
			delta := *trip.Electronic - *s0.Electronic
			fmt.Fprintf(&b, "최적화 S₀(Singlet)/Triplet 상태 Electronic Energy 차이: %s\n", showDelta(delta))
			b.WriteString("주의: 두 상태가 의도한 전자상태인지, Geometry가 수렴했는지, <S²>를 확인해야 합니다. 이 값은 Gibbs Free Energy 차이가 아닙니다.\n")
			if s0.Gibbs != nil && trip.Gibbs != nil {
				fmt.Fprintf(&b, "별도 Frequency 계산에서 얻은 ΔG (Triplet − Singlet): %s\n", showDelta(*trip.Gibbs-*s0.Gibbs))
			}
		}
	} else {
		b.WriteString("S₀/T₁ 비교: 수렴한 Singlet 및 Triplet 최적화 결과가 모두 있을 때 표시됩니다.\n")
	}
	b.WriteString("\n[SOC / BDE / SET / reaction ΔG]\n계산에 필요한 개별 상태나 여러 화학종의 에너지가 존재해야 비교할 수 있습니다. 부재한 수치는 임의 생성하지 않습니다.\n")
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(folder, "analysis.txt"), []byte(b.String()), 0644); err != nil {
		return "", err
	}
	payload := struct {
		Label  string
		Units  map[string]string
		Stages []stageResult
	}{Label: req.Label, Units: map[string]string{"Electronic": "Eh", "Enthalpy": "Eh", "Gibbs": "Eh", "ZPE": "Eh", "EntropyTerm": "Eh (T*S)", "ThermalCorrection": "Eh", "ThermalEnthalpyCorrection": "Eh (H - Electronic)", "Entropy": "Eh/K", "Temperature": "K", "Frequencies": "cm^-1", "Frontier": "eV", "Roots.EV": "eV (Vertical excitation)", "Roots.EH": "Eh (Vertical excitation)", "Roots.WavelengthNM": "nm", "Populations.Spin": "electrons", "Populations.Index": "zero-based ORCA atom index"}, Stages: all}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	if err = os.WriteFile(filepath.Join(folder, "results.json"), data, 0644); err != nil {
		return "", err
	}
	return path, nil
}

// comparisonSettings deliberately requires exact agreement of non-geometry
// settings, including solvent and TDDFT blocks, before subtracting energies.
func comparisonSettings(inp string) string {
	var parts []string
	for _, line := range strings.Split(inp, "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if line == "" || strings.HasPrefix(line, "*") {
			continue
		}
		parts = append(parts, strings.ToLower(strings.Join(strings.Fields(line), " ")))
	}
	return strings.Join(parts, "\n")
}
func analysisAvailability(a OrcaAnalysis, inp string) map[string]string {
	values := map[string]*float64{"Electronic Energy": a.Electronic, "Enthalpy": a.Enthalpy, "Gibbs Free Energy": a.Gibbs, "Zero-Point Energy": a.ZPE, "Thermal Correction": a.ThermalCorrection, "Entropy": a.Entropy, "S2": a.S2}
	result := map[string]string{}
	for name, value := range values {
		if value != nil {
			result[name] = "Available"
		} else if !a.Normal {
			result[name] = "Failed"
		} else {
			result[name] = "Unavailable"
		}
	}
	if !regexp.MustCompile(`(?i)\b(?:Freq|NumFreq|AnFreq)\b`).MatchString(inp) {
		for _, name := range []string{"Enthalpy", "Gibbs Free Energy", "Zero-Point Energy", "Thermal Correction", "Entropy"} {
			if result[name] != "Available" {
				result[name] = "Not calculated"
			}
		}
	}
	if a.Frontier.FoundH && a.Frontier.FoundL {
		result["HOMO/LUMO"] = "Available"
	} else {
		result["HOMO/LUMO"] = "Unavailable"
	}
	if len(a.Roots) > 0 {
		result["Vertical excitation"] = "Available"
	} else if strings.Contains(strings.ToLower(inp), "tddft") {
		result["Vertical excitation"] = "Unavailable"
	} else {
		result["Vertical excitation"] = "Not calculated"
	}
	spinFound := false
	for _, rows := range a.Populations {
		for _, r := range rows {
			if r.Spin != nil {
				spinFound = true
			}
		}
	}
	if spinFound {
		result["Atomic Spin Population"] = "Available"
	} else {
		result["Atomic Spin Population"] = "Unavailable"
	}
	return result
}
