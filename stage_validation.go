package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var failedSCF = regexp.MustCompile(`(?i)SCF (?:NOT CONVERGED|DID NOT CONVERGE)|SCF convergence failure|SCF CONVERGENCE HAS NOT BEEN ACHIEVED`)

func validateStage(folder string, s Stage) error {
	if s.Goat{return validateGOAT(folder,s)}
	inp, err := os.ReadFile(filepath.Join(folder, s.Input))
	if err != nil {
		return err
	}
	outPath := filepath.Join(folder, strings.TrimSuffix(s.Input, filepath.Ext(s.Input))+".out")
	f, err := os.Open(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	a, err := ParseOrcaOutput(f, outPath, string(inp))
	if err != nil {
		return err
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 4*1024*1024)
	var meta strings.Builder
	scfFailure := false
	for scanner.Scan() {
		line := scanner.Text()
		if failedSCF.MatchString(line) {
			scfFailure = true
		}
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "Total Charge") || strings.HasPrefix(trim, "Multiplicity") {
			if meta.Len() < 65536 {
				meta.WriteString(line)
				meta.WriteByte('\n')
			}
		}
	}
	if err = scanner.Err(); err != nil {
		return err
	}
	b := []byte(meta.String())
	if !a.Normal || !a.SCFConverged || a.Electronic == nil || scfFailure {
		return fmt.Errorf("%s: ORCA 정상 종료, SCF 수렴 및 Electronic Energy를 확인하지 못했습니다", s.Name)
	}
	if s.Opt {
		if !a.Optimized {
			return fmt.Errorf("%s: Geometry Optimization이 수렴하지 않았습니다", s.Name)
		}
		xyz, err := os.ReadFile(filepath.Join(folder, stageGeometry(s)))
		if err != nil {
			return fmt.Errorf("최적화 XYZ가 없습니다: %w", err)
		}
		n, z, err := parseXYZAtoms(string(xyz))
		if err != nil {
			return err
		}
		source, err := os.ReadFile(filepath.Join(folder, s.Source))
		if err != nil {
			return err
		}
		sn, sz, err := parseXYZAtoms(string(source))
		if err != nil || n != sn || z != sz || !sameElements(source, xyz) {
			return fmt.Errorf("최적화 XYZ의 원자 수 또는 원소가 입력과 다릅니다")
		}
	}
	if s.Stationary!="" {
        want:=0
        if s.Stationary=="ts"{want=1}
        if a.ImaginaryCount==nil||*a.ImaginaryCount!=want||a.Gibbs==nil{return fmt.Errorf("%s: Frequency/Gibbs 또는 imaginary mode 검증 실패 (필요 %d개)",s.Name,want)}
    }
    if s.HAT=="reactant"||s.HAT=="product"||s.HAT=="ts" {
        want:=0; if s.HAT=="ts" {want=1}
        if a.ImaginaryCount==nil||*a.ImaginaryCount!=want||a.Gibbs==nil {return fmt.Errorf("%s: HAT Gibbs 또는 허수 진동수 검증 실패 (필요 %d개)",s.Name,want)}
    }
    // Only compare explicit output metadata, never the parser's input fallback.
	for _, v := range []struct {
		pattern string
		idx     int
	}{{`(?im)^\s*Total Charge\s*(?:Charge\s*)?\.*\s*([+-]?\d+)`, 1}, {`(?im)^\s*Multiplicity\s*(?:Mult\s*)?\.*\s*(\d+)`, 2}} {
		m := regexp.MustCompile(v.pattern).FindSubmatch(b)
		h := xyzHeader.FindSubmatch(inp)
		if m != nil && h != nil {
			x, _ := strconv.Atoi(string(m[1]))
			y, _ := strconv.Atoi(string(h[v.idx]))
			if x != y {
				return fmt.Errorf("출력 전자상태가 입력 Charge/Multiplicity와 다릅니다")
			}
		}
	}
	return validateHATGeometry(folder,s)
}

// Preserve atom identity/order for spin-index comparisons and downstream stages.
func sameElements(a, b []byte) bool {
	aa := strings.Split(strings.ReplaceAll(string(a), "\r\n", "\n"), "\n")
	bb := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	n, _, err := parseXYZAtoms(string(a))
	if err != nil || len(bb) < n+2 {
		return false
	}
	for i := 2; i < n+2; i++ {
		x := strings.Fields(aa[i])
		y := strings.Fields(bb[i])
		if len(x) == 0 || len(y) == 0 || !strings.EqualFold(x[0], y[0]) {
			return false
		}
	}
	return true
}
