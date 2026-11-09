package main

import (
    "encoding/json"
    "fmt"
    "math"
    "net/http"
    "strings"
)

// Profiles never infer molecular identity or reaction-state roles from filenames.
// The user explicitly assigns calculated structures and confirms stoichiometry.
type ScientificProfileRequest struct {
    JobID string `json:"jobId"`
    Kind string `json:"kind"` // adiabatic or reaction
    Energy string `json:"energy"` // Electronic or Gibbs
    Sources map[string]string `json:"sources"`
    Confirmed bool `json:"confirmed"`
}
type ScientificProfilePoint struct {
    Label string `json:"label"`
    X float64 `json:"x"`
    Y float64 `json:"y"` // relative energy in kJ/mol
    EnergyEh float64 `json:"energyEh"`
    Source string `json:"source"`
}
type ScientificProfileResult struct {
    Kind string `json:"kind"`
    Energy string `json:"energy"`
    Units string `json:"units"`
    Points []ScientificProfilePoint `json:"points"`
    GapKJ *float64 `json:"gapKJ,omitempty"`
    BarrierKJ *float64 `json:"barrierKJ,omitempty"`
    ReactionKJ *float64 `json:"reactionKJ,omitempty"`
    Warnings []string `json:"warnings"`
}
func canonicalMethod(a OrcaAnalysis) string {
    // Only compare the computational level and solvent keyword, not Opt/Freq/OptTS
    // or spin multiplicity. Different method/basis/solvent means no automatic gap.
    drop := map[string]bool{"opt":true,"optts":true,"freq":true,"numfreq":true,
        "sp":true,"tightopt":true,"ukS":true,"uks":true,"rhf":true,"uhf":true,
        "td-dft":true,"tddft":true,"noautostart":true,"printmos":true,"printgap":true}
    var tokens []string
    for _, tok := range strings.Fields(strings.ToLower(strings.TrimSpace(a.Method))) {
        if !drop[tok] {tokens=append(tokens,tok)}
    }
    return strings.Join(tokens, " ")
}
func profileEnergy(a OrcaAnalysis, mode string, isTS bool) (float64,error) {
    if !a.Normal || !a.SCFConverged || !a.Optimized {
        return 0,fmt.Errorf("%s: 정상 종료·SCF 및 Geometry Optimization 수렴이 모두 확인되어야 합니다",a.Filename)
    }
    if mode=="Gibbs" {
        if a.Gibbs==nil || a.Temperature==nil || a.ImaginaryCount==nil {
            return 0,fmt.Errorf("%s: Gibbs/Temperature/Frequency 정보가 완전하지 않습니다",a.Filename)
        }
        expected:=0
        if isTS { expected=1 }
        if *a.ImaginaryCount!=expected {
            return 0,fmt.Errorf("%s: 허수 진동수 %d개 (기대 %d개)",a.Filename,*a.ImaginaryCount,expected)
        }
        if !isFinite(*a.Gibbs) {return 0,fmt.Errorf("%s: 유효하지 않은 Gibbs 값",a.Filename)}
        return *a.Gibbs,nil
    }
    if a.Electronic==nil || !isFinite(*a.Electronic) {
        return 0,fmt.Errorf("%s: 전자 에너지가 없습니다",a.Filename)
    }
    if isTS && (a.ImaginaryCount==nil || *a.ImaginaryCount!=1) {
        return 0,fmt.Errorf("%s: TS는 정확히 하나의 허수 진동수로 검증되어야 합니다",a.Filename)
    }
    return *a.Electronic,nil
}
func isFinite(v float64)bool {return !math.IsNaN(v)&&!math.IsInf(v,0)}
func scientificProfile(req ScientificProfileRequest, sources map[string]OrcaAnalysis) (ScientificProfileResult,error) {
    result:=ScientificProfileResult{Kind:req.Kind,Energy:req.Energy,Units:"kJ/mol",Points:[]ScientificProfilePoint{},Warnings:[]string{}}
    if !req.Confirmed {return result,fmt.Errorf("사용자가 동일한 원자 구성/반응식과 계산 조건을 확인해야 합니다")}
    if req.Kind!="adiabatic"&&req.Kind!="reaction" {return result,fmt.Errorf("지원하지 않는 Profile 종류")}
    if req.Energy!="Electronic"&&req.Energy!="Gibbs" {return result,fmt.Errorf("Energy는 Electronic 또는 Gibbs로 선택하세요")}
    roles:=[]string{"S0","T1"}
    if req.Kind=="reaction" {roles=[]string{"Reactant","TS","Product"}}
    ordered:=make([]OrcaAnalysis,0,len(roles))
    seen:=map[string]bool{}
    for _,role:=range roles {
        filename:=req.Sources[role]
        if filename==""||seen[filename] {return result,fmt.Errorf("%s: 겹치지 않는 결과 파일을 지정하세요",role)}
        seen[filename]=true
        a,ok:=sources[filename]
        if !ok {return result,fmt.Errorf("%s: 해당 결과 파일을 찾지 못했습니다",role)}
        if a.Method=="" || a.Geometry=="" || canonicalMethod(a)=="" {
            return result,fmt.Errorf("%s: 원본 INP의 Method/Geometry 정보가 필요합니다",filename)
        }
        ordered=append(ordered,a)
    }
    ref:=ordered[0]
    for i,a:=range ordered {
        if canonicalMethod(a)!=canonicalMethod(ref) {
            return result,fmt.Errorf("%s: Functional/Basis/Dispersion/Solvent/SCF 설정이 다른 결과는 직접 비교하지 않습니다",roles[i])
        }
        if a.Charge!=ref.Charge {return result,fmt.Errorf("%s: Charge가 서로 다릅니다",roles[i])}
        if req.Energy=="Gibbs" && (ref.Temperature==nil||a.Temperature==nil||math.Abs(*a.Temperature-*ref.Temperature)>0.5) {
            return result,fmt.Errorf("%s: Gibbs 계산의 온도를 일치시켜야 합니다",roles[i])
        }
    }
    if req.Kind=="adiabatic" {
        if ordered[0].Multiplicity!=1||ordered[1].Multiplicity!=3 {
            return result,fmt.Errorf("Adiabatic S0/T1 비교에는 Singlet(1) 및 Triplet(3) 최적화 계산이 필요합니다")
        }
        result.Warnings=append(result.Warnings,
            "두 구조의 동일 분자 여부, Triplet 상태 정체성 및 UKS <S²>는 사용자가 검증해야 합니다.",
            "이 값은 최적화 구조 간 Adiabatic 차이이며 S0 구조의 TD-DFT Vertical T1과 다릅니다.")
    } else {
        if ordered[0].Multiplicity!=ordered[1].Multiplicity||ordered[0].Multiplicity!=ordered[2].Multiplicity {
            return result,fmt.Errorf("Reaction Profile은 동일한 Spin Surface의 구조만 자동 비교합니다")
        }
        if ordered[1].ImaginaryCount==nil||*ordered[1].ImaginaryCount!=1 {
            return result,fmt.Errorf("TS 검증용 Frequency (허수 모드 정확히 하나)가 필요합니다")
        }
        result.Warnings=append(result.Warnings,
            "원자 조성/반응식/표준상태/TS 연결성(IRC)과 계산 수준 일치는 사용자 확인 사항입니다.",
            "다분자 반응물 및 생성물의 에너지 합산은 이 3-구조 Profile에 포함되지 않습니다.")
    }
    values:=make([]float64,len(ordered))
    for i,a:=range ordered {
        n,e:=profileEnergy(a,req.Energy,req.Kind=="reaction"&&i==1)
        if e!=nil{return result,e}
        values[i]=n
        result.Points=append(result.Points,ScientificProfilePoint{
            Label:roles[i],X:float64(i+1),Y:(n-values[0])*ehToKJ,
            EnergyEh:n,Source:req.Sources[roles[i]],
        })
    }
    if req.Kind=="adiabatic" {
        gap:=(values[1]-values[0])*ehToKJ
        result.GapKJ=&gap
    } else {
        barrier:=(values[1]-values[0])*ehToKJ
        reaction:=(values[2]-values[0])*ehToKJ
        result.BarrierKJ=&barrier
        result.ReactionKJ=&reaction
    }
    return result,nil
}
func (a *App) scientificProfileRoute(w http.ResponseWriter,r *http.Request) {
    if r.Method!=http.MethodPost {fail(w,405,"POST만 지원합니다");return}
    var req ScientificProfileRequest
    if json.NewDecoder(http.MaxBytesReader(w,r.Body,16<<10)).Decode(&req)!=nil {
        fail(w,400,"Profile 요청 JSON을 읽지 못했습니다");return
    }
    if len(req.Sources)>3 {fail(w,400,"최대 3개 결과만 비교할 수 있습니다");return}
    folder:=""
    if req.JobID!=""&&a.queue!=nil {
        a.queue.mu.Lock()
        for _,e:=range a.queue.Jobs {
            if e.ID==req.JobID&&e.Job!=nil {folder=e.Job.Folder;break}
        }
        a.queue.mu.Unlock()
    }else if req.JobID=="" {
        a.mu.Lock()
        if a.job!=nil {folder=a.job.Folder}
        a.mu.Unlock()
    }
    if folder=="" {fail(w,404,"선택된 Job 폴더가 없습니다");return}
    v,err:=loadVisualization(folder)
    if err!=nil {fail(w,400,err.Error());return}
    src:=map[string]OrcaAnalysis{}
    for _,s:=range v.Sources {
        if strings.EqualFold(filepathExt(s.Name),".out") {src[s.Name]=s.Analysis}
    }
    res,err:=scientificProfile(req,src)
    if err!=nil {fail(w,422,err.Error());return}
    respond(w,200,res)
}
func filepathExt(s string)string {idx:=strings.LastIndex(s,".");if idx<0{return ""};return s[idx:]}
