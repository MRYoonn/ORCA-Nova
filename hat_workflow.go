package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var hatEndFile = regexp.MustCompile(`(?im)^\s*NEB_END_XYZFILE\s+"([^"\r\n]+)"\s*$`)
var hatTemp = regexp.MustCompile(`(?im)^\s*Temp\s+([0-9.]+)\s*$`)

// Every stage is one stationary-state workflow, not a separate batch molecule.
func validateHATRequest(req RunRequest, files map[string]string) error {
	present := false
	for _, s := range req.Stages { if s.HAT != "" || s.EndSource != "" {present = true} }
	if !present { return nil }
	if len(req.Stages) != 4 {return fmt.Errorf("HAT: R/P/TS/IRC 네 단계가 필요합니다")}
	roles := []string{"reactant","product","ts","irc"}
	var level []string
	var temperature string
	var q,m int
	for i,s := range req.Stages {
		if s.HAT != roles[i] || s.Opt != (i<3) || s.Goat || s.Stationary!="" || s.CompositeFrom!="" || (i!=2 && s.EndSource!="") {
			return fmt.Errorf("HAT: 단계 역할 또는 산출물 선언 오류")
		}
		inp:=files[strings.ToLower(s.Input)]
		cq,cm,e:=chargeMultiplicity(inp)
		if e!=nil {return fmt.Errorf("HAT: 전하/Multiplicity 누락")}
		if i==0 {q,m=cq,cm} else if q!=cq||m!=cm {return fmt.Errorf("HAT: 전하/Multiplicity 불일치")}
		k:=keywordLine(inp)
		if k["opt"] != (i<2) || k["neb-ts"] != (i==2) || k["irc"] != (i==3) || k["freq"] != (i<3) || k["optts"] || k["numfreq"] {
			return fmt.Errorf("HAT: 단계별 Opt/Freq/NEB-TS/IRC 키워드 불일치")
		}
		common:=[]string{}
		for name:=range k {switch name {case "opt","freq","neb-ts","irc":default:common=append(common,name)}}
		slices.Sort(common)
		if i==0 {level=common} else if !slices.Equal(common,level) {return fmt.Errorf("HAT: 계산 수준 또는 용매 불일치")}
		if i<3 {
			t:=hatTemp.FindStringSubmatch(inp)
			if t==nil {return fmt.Errorf("HAT: Frequency 온도를 명시하세요")}
			if i==0 {temperature=t[1]} else if t[1]!=temperature {return fmt.Errorf("HAT: Frequency 온도 불일치")}
		}
	}
	r,p,t,irc:=req.Stages[0],req.Stages[1],req.Stages[2],req.Stages[3]
	if strings.EqualFold(r.Source,p.Source) {return fmt.Errorf("HAT: 서로 다른 두 XYZ가 필요합니다")}
	a,b:=files[strings.ToLower(r.Source)],files[strings.ToLower(p.Source)]
	n,z,e:=parseXYZAtoms(a); pn,pz,pe:=parseXYZAtoms(b)
	if e!=nil||pe!=nil||n!=pn||z!=pz||!sameElements([]byte(a),[]byte(b)) {return fmt.Errorf("HAT: R/P XYZ 원자 수 또는 원소 순서 불일치")}
	if !strings.EqualFold(t.Source,stageGeometry(r)) || !strings.EqualFold(t.EndSource,stageGeometry(p)) || !strings.EqualFold(irc.Source,stageGeometry(t)) {
		return fmt.Errorf("HAT: 최적화 XYZ 연결 오류")
	}
	ref:=hatEndFile.FindStringSubmatch(files[strings.ToLower(t.Input)])
	if ref==nil||!safeName(ref[1])||!strings.EqualFold(ref[1],t.EndSource) {return fmt.Errorf("HAT: NEB_END_XYZFILE 연결 오류")}
	return nil
}

func hatWorkflowReport(folder string,req RunRequest,completed []string) string {
	if len(req.Stages)!=4||req.Stages[0].HAT!="reactant" {return ""}
	var b strings.Builder
	b.WriteString("\n[HAT / NEB-TS Gibbs barrier]\n")
	files:=map[string]string{}
	for _,f:=range req.Files {files[strings.ToLower(f.Name)]=f.Content}
	if e:=validateHATRequest(req,files);e!=nil {fmt.Fprintf(&b,"Unavailable: %s\n",e);return b.String()}
	var g [3]float64
	for i,s:=range req.Stages[:3] {
		if !slices.Contains(completed,s.Input) {fmt.Fprintf(&b,"Unavailable: %s 단계 미완료\n",s.Name);return b.String()}
		if err:=validateStage(folder,s);err!=nil {fmt.Fprintf(&b,"Unavailable: %s\n",err);return b.String()}
		inp,err:=os.ReadFile(filepath.Join(folder,s.Input))
		if err!=nil {fmt.Fprintf(&b,"Unavailable: %s\n",err);return b.String()}
		path:=filepath.Join(folder,strings.TrimSuffix(s.Input,filepath.Ext(s.Input))+".out")
		f,err:=os.Open(path)
		if err!=nil {fmt.Fprintf(&b,"Unavailable: %s\n",err);return b.String()}
		a,err:=ParseOrcaOutput(f,path,string(inp));f.Close()
		if err!=nil||a.Gibbs==nil||math.IsNaN(*a.Gibbs)||math.IsInf(*a.Gibbs,0) {fmt.Fprintf(&b,"Unavailable: %s Gibbs\n",s.Name);return b.String()}
		g[i]=*a.Gibbs
		fmt.Fprintf(&b,"%s G = %.10f Eh\n",s.Name,g[i])
	}
	fmt.Fprintf(&b,"Reaction ΔG = %.4f kcal/mol\nForward ΔG‡ = %.4f kcal/mol\nReverse ΔG‡ = %.4f kcal/mol\n",(g[1]-g[0])*627.509474,(g[2]-g[0])*627.509474,(g[2]-g[1])*627.509474)
	if g[2]<g[0]||g[2]<g[1] {b.WriteString("Warning: TS free energy below an endpoint; inspect thermal corrections and connectivity.\n")}
	if slices.Contains(completed,req.Stages[3].Input) {b.WriteString("IRC execution completed; endpoint identity NOT automatically verified.\n")} else {b.WriteString("IRC not completed; connectivity unverified.\n")}
	b.WriteString("Provisional barriers only: inspect the single imaginary mode, IRC endpoints, transferring H atom mapping, spin populations, <S²> and conformers.\n")
	return b.String()
}
