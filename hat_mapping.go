package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type hatMapping struct {
	H        int `json:"h"`
	Donor    int `json:"donor"`
	Acceptor int `json:"acceptor"`
}

type hatBondChange struct {
	A      int  `json:"a"`
	B      int  `json:"b"`
	Formed bool `json:"formed"`
}

type hatMappingReview struct {
	Valid            bool            `json:"valid"`
	Changes          []hatBondChange `json:"changes"`
	SwapCandidates   []int           `json:"swapCandidates"`
	CorrectedProduct string          `json:"correctedProduct,omitempty"`
	Atoms            []string        `json:"atoms"`
	Reactant         []tripVec       `json:"reactant"`
	Product          []tripVec       `json:"product"`
	Bonds            [][2]int        `json:"bonds"`
	Aligned          bool            `json:"aligned"`
}

func hatEndpoints(reactant, product string) (tripMolecule, tripMolecule, error) {
	r, err := parseTripletXYZ(reactant)
	if err != nil {
		return r, tripMolecule{}, fmt.Errorf("Reactant XYZ: %w", err)
	}
	p, err := parseTripletXYZ(product)
	if err != nil {
		return r, p, fmt.Errorf("Product XYZ: %w", err)
	}
	if !slices.Equal(r.Atoms, p.Atoms) {
		return r, p, fmt.Errorf("Reactant/Product XYZ의 원소 순서 또는 원자 수가 다릅니다")
	}
	return r, p, nil
}

func hatChanges(r, p tripMolecule, exchangeA, exchangeB int) []hatBondChange {
	index := func(i int) int {
		switch i {
		case exchangeA:
			return exchangeB
		case exchangeB:
			return exchangeA
		}
		return i
	}
	changes := []hatBondChange{}
	for i := range r.Atoms {
		for j := range i {
			rBond, pBond := r.Bond[i][j], p.Bond[index(i)][index(j)]
			if rBond != pBond {
				changes = append(changes, hatBondChange{A: j, B: i, Formed: pBond})
			}
		}
	}
	return changes
}

func (m hatMapping) matches(changes []hatBondChange) bool {
	if len(changes) != 2 {
		return false
	}
	lost, gained := false, false
	for _, c := range changes {
		lost = lost || (!c.Formed && c.A == min(m.H, m.Donor) && c.B == max(m.H, m.Donor))
		gained = gained || (c.Formed && c.A == min(m.H, m.Acceptor) && c.B == max(m.H, m.Acceptor))
	}
	return lost && gained
}

func (m hatMapping) checkIndices(r tripMolecule) error {
	if m.H < 0 || m.Donor < 0 || m.Acceptor < 0 || max(m.H, m.Donor, m.Acceptor) >= len(r.Atoms) ||
		r.Atoms[m.H] != "H" || r.Atoms[m.Donor] == "H" || r.Atoms[m.Acceptor] == "H" || m.Donor == m.Acceptor {
		return fmt.Errorf("이동 H, donor, acceptor의 0-based 원자 번호를 확인하세요")
	}
	if neighbors := tripNeighbors(r, m.H, ""); len(neighbors) != 1 || neighbors[0] != m.Donor {
		return fmt.Errorf("Reactant의 H%d가 donor %d에만 결합해야 합니다", m.H, m.Donor)
	}
	return nil
}

func reviewHATMapping(reactant, product string, mapping hatMapping) (hatMappingReview, error) {
	r, p, err := hatEndpoints(reactant, product)
	if err != nil {
		return hatMappingReview{}, err
	}
	if err := mapping.checkIndices(r); err != nil {
		return hatMappingReview{}, err
	}
	review := hatMappingReview{Atoms: r.Atoms, Reactant: r.XYZ, Product: p.XYZ,
		Changes: hatChanges(r, p, -1, -1), SwapCandidates: []int{}}
	review.Valid = mapping.matches(review.Changes)
	if !review.Valid {
		for i, atom := range p.Atoms {
			if i != mapping.H && atom == "H" && mapping.matches(hatChanges(r, p, mapping.H, i)) {
				review.SwapCandidates = append(review.SwapCandidates, i)
			}
		}
		if len(review.SwapCandidates) == 1 {
			lines := strings.Split(product, "\n")
			a, b := mapping.H+2, review.SwapCandidates[0]+2
			lines[a], lines[b] = lines[b], lines[a]
			review.CorrectedProduct = strings.Join(lines, "\n")
		}
	}
	for i := range r.Atoms {
		for j := range i {
			if r.Bond[i][j] {
				review.Bonds = append(review.Bonds, [2]int{j, i})
			}
		}
	}
	// Match donor→acceptor and a shared off-axis heavy atom. This is a
	// visual alignment of indexed anchors, not a claim of unique atom identity.
	rOrigin, pOrigin := r.XYZ[mapping.Donor], p.XYZ[mapping.Donor]
	rX, errR := vecUnit(vecSub(r.XYZ[mapping.Acceptor], rOrigin))
	pX, errP := vecUnit(vecSub(p.XYZ[mapping.Acceptor], pOrigin))
	if errR == nil && errP == nil {
		bestScore := 0.0
		var rZ, pZ tripVec
		for i, atom := range r.Atoms {
			if atom == "H" || i == mapping.Donor || i == mapping.Acceptor {
				continue
			}
			rTrial := vecCross(rX, vecSub(r.XYZ[i], rOrigin))
			pTrial := vecCross(pX, vecSub(p.XYZ[i], pOrigin))
			score := min(vecNorm(rTrial), vecNorm(pTrial))
			if score > bestScore {
				bestScore, rZ, pZ = score, rTrial, pTrial
			}
		}
		if bestScore > 1e-6 {
			rZ, _ = vecUnit(rZ)
			pZ, _ = vecUnit(pZ)
			rY, pY := vecCross(rZ, rX), vecCross(pZ, pX)
			project := func(points []tripVec, origin, x, y, z tripVec) []tripVec {
				local := make([]tripVec, len(points))
				for i, point := range points {
					v := vecSub(point, origin)
					local[i] = tripVec{vecDot(v, x), vecDot(v, y), vecDot(v, z)}
				}
				return local
			}
			review.Reactant = project(r.XYZ, rOrigin, rX, rY, rZ)
			review.Product = project(p.XYZ, pOrigin, pX, pY, pZ)
			review.Aligned = true
		}
	}
	return review, nil
}

func hatMappingRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 405, "POST required")
		return
	}
	var req struct {
		Reactant string      `json:"reactant"`
		Product  string      `json:"product"`
		Mapping  *hatMapping `json:"mapping"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&req); err != nil || req.Mapping == nil {
		fail(w, 400, "Reactant, Product 및 원자 번호가 필요합니다")
		return
	}
	review, err := reviewHATMapping(req.Reactant, req.Product, *req.Mapping)
	if err != nil {
		fail(w, 422, err.Error())
		return
	}
	respond(w, 200, review)
}

func validateHATGeometry(folder string, stage Stage) error {
	if stage.HAT != "reactant" && stage.HAT != "product" {
		return nil
	}
	reference, err := readGloriusFile(folder, stage.Source)
	if err != nil {
		return err
	}
	optimized, err := readGloriusFile(folder, stageGeometry(stage))
	if err != nil {
		return err
	}
	r, p, err := hatEndpoints(reference, optimized)
	if err != nil {
		return err
	}
	if changes := hatChanges(r, p, -1, -1); len(changes) != 0 {
		return fmt.Errorf("%s: Opt 이후 결합/원자 대응 변경 %v", stage.Name, changes)
	}
	return nil
}

func hatIRCEndpoints(folder string, req RunRequest) string {
	s := req.Stages
	stem := strings.TrimSuffix(s[3].Input, filepath.Ext(s[3].Input))
	var molecules []tripMolecule
	for _, name := range []string{stageGeometry(s[0]), stageGeometry(s[1]), stem + "_IRC_F.xyz", stem + "_IRC_B.xyz"} {
		raw, err := os.ReadFile(filepath.Join(folder, name))
		if err != nil {
			return "Forward/backward 최종 IRC XYZ를 확인할 수 없습니다."
		}
		m, err := parseTripletXYZ(string(raw))
		if err != nil {
			return "IRC endpoint XYZ 검사 실패: " + err.Error()
		}
		molecules = append(molecules, m)
	}
	matches := func(endpoint, reference int) bool {
		a, b := molecules[endpoint], molecules[reference]
		return slices.Equal(a.Atoms, b.Atoms) && len(hatChanges(a, b, -1, -1)) == 0
	}
	switch {
	case matches(2, 0) && matches(3, 1):
		return "Forward → Reactant, Backward → Product (inferred connectivity)"
	case matches(2, 1) && matches(3, 0):
		return "Forward → Product, Backward → Reactant (inferred connectivity)"
	default:
		return "Forward/backward 결합 연결성이 지정한 Reactant/Product 쌍과 일치하지 않습니다."
	}
}
