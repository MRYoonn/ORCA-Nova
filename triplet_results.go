package main

import (
 "encoding/csv"
 "encoding/json"
 "fmt"
 "io"
 "math"
 "net/http"
 "os"
 "path/filepath"
 "sort"
 "strconv"
 "strings"
)

// Evidence-based output. Never infer a SOMO or T1 identity from a successful
// optimization alone. Every chemical role is tied to a graph-derived index.
type tripRoleSpin struct{
 Role string `json:"role"`
 Index int `json:"index"`
 Element string `json:"element"`
 Mulliken *float64 `json:"mulliken"`
 Loewdin *float64 `json:"loewdin"`
 Hirshfeld *float64 `json:"hirshfeld"`
}
type tripResult struct{
 Target string `json:"target"`
 Method string `json:"method"`
 MethodFingerprint string `json:"methodFingerprint"`
 SOMOCandidates []tripSOMO `json:"somoCandidates"`
 Charge int `json:"charge"`
 Multiplicity int `json:"multiplicity"`
 Converged bool `json:"converged"`
 SCFConverged bool `json:"scfConverged"`
 OptimizationConverged bool `json:"optimizationConverged"`
 MinimumVerified bool `json:"minimumVerified"`
 FrequencyChecked bool `json:"frequencyChecked"`
 ImaginaryCount *int `json:"imaginaryCount"`
 EnergyEh *float64 `json:"energyEh"`
 S2 *float64 `json:"s2"`
 Roles tripRoles `json:"roles"`
 Spin []tripRoleSpin `json:"spin"`
 Bonds map[string]float64 `json:"bonds"`
 FinalXYZ string `json:"finalXYZ,omitempty"`
 CubeAvailable bool `json:"cubeAvailable"`
 GBWAvailable bool `json:"gbwAvailable"`
 Warnings []string `json:"warnings"`
}
type tripCompareRequest struct{
 JobID string `json:"jobId"`
 OriginalXYZ string `json:"originalXYZ"`
 OriginalOUT string `json:"originalOut"`
 OriginalINP string `json:"originalInp"`
 OriginalRoles tripRoles `json:"originalRoles,omitempty"`
 S0OUT string `json:"s0Out,omitempty"`
 S0INP string `json:"s0Inp,omitempty"`
 S0XYZ string `json:"s0Xyz,omitempty"`
}
type tripComparison struct{
 Original tripResult `json:"original"`
 Transfer tripResult `json:"transfer"`
 SameTarget bool `json:"sameTarget"`
 DeltaEnergyKcal *float64 `json:"deltaEnergyKcal"`
 DeltaEnergyEh *float64 `json:"deltaEnergyEh"`
 ExcitationKcal *float64 `json:"excitationKcal"`
 Warnings []string `json:"warnings"`
}
func tripPop(a OrcaAnalysis,scheme string,index int)*float64{
 for _,entry:=range a.Populations[scheme]{
  if entry.Index==index&&entry.Spin!=nil{v:=*entry.Spin;return &v}
 }
 return nil
}
func tripInspect(m tripMolecule,roles tripRoles,analysis OrcaAnalysis,finalXYZ,target string,frequency bool)tripResult{
 result:=tripResult{
  Target:target,Method:analysis.Method,Charge:analysis.Charge,Multiplicity:analysis.Multiplicity,
  Converged:analysis.Normal&&analysis.SCFConverged&&analysis.Electronic!=nil,
  SCFConverged:analysis.SCFConverged,OptimizationConverged:analysis.Optimized,
  EnergyEh:analysis.Electronic,S2:analysis.S2,Roles:roles,FinalXYZ:finalXYZ,
  Bonds:map[string]float64{},Spin:[]tripRoleSpin{},Warnings:[]string{},
  FrequencyChecked:frequency,ImaginaryCount:analysis.ImaginaryCount,
 }
 if frequency&&analysis.ImaginaryCount!=nil&&*analysis.ImaginaryCount==0{result.MinimumVerified=true}
 if !frequency {result.Warnings=append(result.Warnings,"Frequency 미계산: 진동수로 Triplet 최소점을 검증하지 않았습니다.")}
 if frequency&&!result.MinimumVerified{result.Warnings=append(result.Warnings,"허수 진동수가 0개인지 검증되지 않았습니다.")}
 if analysis.S2==nil{result.Warnings=append(result.Warnings,"⟨S²⟩ 값을 OUT에서 확인할 수 없습니다.")}
 if analysis.S2!=nil&&math.Abs(*analysis.S2-2.0)>0.10{result.Warnings=append(result.Warnings,fmt.Sprintf("Triplet 기대 ⟨S²⟩=2와 차이가 있습니다: %.4f",*analysis.S2))}
 result.Warnings=append(result.Warnings,"UKS Triplet 계산의 정상 종료가 최저 T₁ 상태/동일 전자배치의 증거는 아닙니다. SOMO, 안정성, 상태 성격을 별도로 검증하세요.")
 if err:=tripCheckRoles(m,roles);err!=nil{result.Warnings=append(result.Warnings,"원자 역할 매핑 오류: "+err.Error());return result}
 for _,role:=range []string{"C2","C5","C6"}{
  idx:=roles[role]
  result.Spin=append(result.Spin,tripRoleSpin{
   Role:role,Index:idx,Element:m.Atoms[idx],
   Mulliken:tripPop(analysis,"Mulliken",idx),
   Loewdin:tripPop(analysis,"Loewdin",idx),
   Hirshfeld:tripPop(analysis,"Hirshfeld",idx),
  })
 }
 for _,pair:=range [][2]string{{"C2","N1"},{"N1","C6"},{"C5","C6"},{"C2","S2"}}{
  key:=pair[0]+"-"+pair[1]
  result.Bonds[key]=tripRoleDistance(m,roles,pair[0],pair[1])
 }
 for _,key:=range []string{"Mulliken","Loewdin","Hirshfeld"}{
  if len(analysis.Populations[key])==0{result.Warnings=append(result.Warnings,key+" 원자별 Spin Population 미검출 · Not calculated")}
 }
 return result
}
func tripReadResult(folder string,req RunRequest)(tripResult,error){
 result:=tripResult{Warnings:[]string{"완료 결과가 없습니다."}}
 if req.Triplet==nil {return result,fmt.Errorf("Triplet Geometry Transfer Job이 아닙니다")}
 meta:=req.Triplet
 result.Target=meta.TargetName
 result.Roles=meta.Roles
 xyzPath:=filepath.Join(folder,"step2_triplet.xyz")
 outPath:=filepath.Join(folder,"step3_spin.out")
 inpPath:=filepath.Join(folder,"step3_spin.inp")
 xyz,xe:=os.ReadFile(xyzPath);out,oe:=os.ReadFile(outPath);inp,ie:=os.ReadFile(inpPath)
 if xe!=nil||oe!=nil||ie!=nil{return result,fmt.Errorf("Triplet 최종 XYZ/OUT/INP가 아직 모두 생성되지 않았습니다")}
 m,err:=parseTripletXYZ(string(xyz));if err!=nil{return result,err}
 a,err:=ParseOrcaOutput(strings.NewReader(string(out)),outPath,string(inp));if err!=nil{return result,err}
 result=tripInspect(m,meta.Roles,a,string(xyz),meta.TargetName,meta.Frequency)
 result.MethodFingerprint=tripMethodFingerprint(string(inp))
 result.SOMOCandidates=tripSOMOCandidates(string(out))
 if len(result.SOMOCandidates)==0{result.Warnings=append(result.Warnings,"UKS α/β orbital occupations에서 SOMO 후보를 식별하지 못했습니다. GBW와 OUT을 확인하세요.")}
 optOut,err:=os.ReadFile(filepath.Join(folder,"step2_triplet.out"))
 if err==nil{
  optInp,_:=os.ReadFile(filepath.Join(folder,"step2_triplet.inp"))
  opt,parseErr:=ParseOrcaOutput(strings.NewReader(string(optOut)),"step2_triplet.out",string(optInp))
  if parseErr==nil{
   result.OptimizationConverged=opt.Normal&&opt.SCFConverged&&opt.Optimized
   if meta.Frequency{
    result.ImaginaryCount=opt.ImaginaryCount
    result.MinimumVerified=opt.ImaginaryCount!=nil&&*opt.ImaginaryCount==0
   }
  }
 }
 result.CubeAvailable=tripNonempty(filepath.Join(folder,"step3_spin.cube"))
 result.GBWAvailable=tripNonempty(filepath.Join(folder,"step2_triplet.gbw"))
 if !result.CubeAvailable{result.Warnings=append(result.Warnings,"Step 3 Spin Density Cube가 생성되지 않았습니다")}
 if !result.GBWAvailable{result.Warnings=append(result.Warnings,"Step 2 Triplet GBW가 생성되지 않았습니다")}
 result.Converged=result.Converged&&result.OptimizationConverged&&result.CubeAvailable&&result.GBWAvailable
 return result,nil
}
func tripNonempty(path string)bool{st,err:=os.Stat(path);return err==nil&&!st.IsDir()&&st.Size()>0}
func validateTripletStage(folder string,stage Stage)error{
 switch stage.Input{
 case "step2_triplet.inp":
  if !tripNonempty(filepath.Join(folder,"step2_triplet.gbw")){return fmt.Errorf("STEP 2 GBW 파일이 생성되지 않았습니다")}
 case "step3_spin.inp":
  if !tripNonempty(filepath.Join(folder,"step3_spin.cube")){return fmt.Errorf("STEP 3 Spin Density CUBE가 생성되지 않았습니다")}
  inp,_:=os.ReadFile(filepath.Join(folder,stage.Input))
  f,err:=os.Open(filepath.Join(folder,"step3_spin.out"));if err!=nil{return err}
  defer f.Close()
  a,err:=ParseOrcaOutput(f,"step3_spin.out",string(inp));if err!=nil{return err}
  if a.S2==nil{return fmt.Errorf("최종 ⟨S²⟩ 출력이 없습니다. 실제 Triplet 전자상태를 검토하세요")}
  if len(a.Populations["Mulliken"])==0||len(a.Populations["Loewdin"])==0{return fmt.Errorf("Mulliken 또는 Löwdin Spin Population 출력이 없습니다")}
 }
 return nil
}
func writeTripletSummary(folder string,req RunRequest)error{
 if req.Triplet==nil{return nil}
 result,err:=tripReadResult(folder,req)
 if err!=nil{return nil} // Incomplete stages must never invent energy or spin values.
 data,err:=json.MarshalIndent(result,"","  ");if err!=nil{return err}
 if err=os.WriteFile(filepath.Join(folder,"triplet_spin_report.json"),data,0600);err!=nil{return err}
 f,err:=os.Create(filepath.Join(folder,"triplet_spin_report.csv"));if err!=nil{return err}
 defer f.Close()
 w:=csv.NewWriter(f)
 _=w.Write([]string{"Target","Role","0-based index","Mulliken","Loewdin","Hirshfeld","Energy Eh","S2","Converged","Minimum Verified"})
 for _,row:=range result.Spin{
  _=w.Write([]string{csvSafe(result.Target),row.Role,strconv.Itoa(row.Index),tripFloat(row.Mulliken),tripFloat(row.Loewdin),tripFloat(row.Hirshfeld),tripFloat(result.EnergyEh),tripFloat(result.S2),strconv.FormatBool(result.Converged),strconv.FormatBool(result.MinimumVerified)})
 }
 w.Flush();return w.Error()
}
func tripFloat(v *float64)string{if v==nil{return "Not calculated"};return strconv.FormatFloat(*v,'g',11,64)}
func tripSameFormula(a,b tripMolecule)bool{
 if len(a.Atoms)!=len(b.Atoms){return false}
 x,y:=map[string]int{},map[string]int{}
 for _,v:=range a.Atoms{x[v]++};for _,v:=range b.Atoms{y[v]++}
 for k,v:=range x{if y[k]!=v{return false}}
 return true
}
// Graph fingerprint refines each atomic environment including anchored ring
// labels. It is a conservative test, not proof of unique constitutional identity.
func tripGraphFingerprint(m tripMolecule,role tripRoles)string{
 labels:=make([]string,len(m.Atoms))
 for i,x:=range m.Atoms{labels[i]=x}
 for k,v:=range role{if v>=0&&v<len(labels){labels[v]+=":"+k}}
 for round:=0;round<6;round++{
  next:=make([]string,len(labels))
  for i:=range labels{
   neighborhood:=[]string{}
   for j,b:=range m.Bond[i]{if b{neighborhood=append(neighborhood,labels[j])}}
   sort.Strings(neighborhood)
   // Short deterministic strings; avoid exponential size growth.
   h:=uint64(1469598103934665603)
   for _,ch:=range labels[i]+"["+strings.Join(neighborhood,",")+"]"{
    h^=uint64(ch);h*=1099511628211
   }
   next[i]=fmt.Sprintf("%016x",h)
  }
  labels=next
 }
 sort.Strings(labels);return strings.Join(labels,",")
}
func tripCompareResult(originalXYZ,originalOUT,originalINP string,roles tripRoles,transfer tripResult,frequency bool)(tripComparison,error){
 c:=tripComparison{Warnings:[]string{}}
 m,err:=parseTripletXYZ(originalXYZ);if err!=nil{return c,fmt.Errorf("Original XYZ: %w",err)}
 if len(roles)==0{
  candidates,e:=tripTryRoles(m);if e!=nil{return c,e}
  if len(candidates)!=1{return c,fmt.Errorf("Original XYZ의 링 매핑이 모호합니다. 수동으로 지정하세요")}
  roles=candidates[0]
 }
 if err=tripCheckRoles(m,roles);err!=nil{return c,err}
 if originalINP==""||originalOUT==""{return c,fmt.Errorf("Original W2 OUT와 INP가 필요합니다")}
 a,err:=ParseOrcaOutput(strings.NewReader(originalOUT),"original.out",originalINP);if err!=nil{return c,err}
 c.Original=tripInspect(m,roles,a,originalXYZ,"Original",frequency)
 c.Original.MethodFingerprint=tripMethodFingerprint(originalINP)
 c.Original.SOMOCandidates=tripSOMOCandidates(originalOUT)
 c.Transfer=transfer
 if transfer.FinalXYZ==""{return c,fmt.Errorf("Transfer 최종 XYZ가 누락됐습니다")}
 t,err:=parseTripletXYZ(transfer.FinalXYZ);if err!=nil{return c,err}
 if !tripSameFormula(m,t)||a.Charge!=transfer.Charge||a.Multiplicity!=transfer.Multiplicity {
  return c,fmt.Errorf("Original/Transfer의 화학식 또는 Charge/Multiplicity가 다릅니다. 서로 다른 기질의 절대 에너지 비교 금지")
 }
 if tripGraphFingerprint(m,roles)!=tripGraphFingerprint(t,transfer.Roles){
  return c,fmt.Errorf("Original과 Transfer의 원자 연결성/고리 역할이 일치하지 않습니다")
 }
 c.SameTarget=true
 if c.Original.MethodFingerprint==""||transfer.MethodFingerprint==""||c.Original.MethodFingerprint!=transfer.MethodFingerprint {
  c.Warnings=append(c.Warnings,"Functional/Basis/Dispersion/Solvent/SCF 설정이 다릅니다. 상대 에너지를 산출하지 않습니다.")
 }else if a.Electronic!=nil&&transfer.EnergyEh!=nil&&a.Normal&&a.SCFConverged&&a.Optimized&&transfer.Converged&&transfer.OptimizationConverged {
  delta:=*transfer.EnergyEh-*a.Electronic;deltaKcal:=delta*ehToKcal
  c.DeltaEnergyEh=&delta;c.DeltaEnergyKcal=&deltaKcal
 }else{c.Warnings=append(c.Warnings,"최종 에너지 또는 수렴 조건이 미확인이라 ΔE를 계산하지 않습니다")}
 c.Warnings=append(c.Warnings,"동일 Target의 UKS 구조 간 ΔE만 비교했습니다. 이는 Excitation Energy도, 동일 T₁ 상태의 증명도 아닙니다.")
 return c,nil
}
func (a *App) tripletExtraRoute(w http.ResponseWriter,r *http.Request){
 switch r.URL.Path{
 case "/api/triplet/result":
  if r.Method!=http.MethodGet{fail(w,405,"GET만 허용합니다");return}
  id:=r.URL.Query().Get("id")
  var folder string;var req RunRequest
  if a.queue!=nil{
   a.queue.mu.Lock()
   for _,e:=range a.queue.Jobs{if e.ID==id&&e.Job!=nil&&e.Request.Triplet!=nil{folder=e.Job.Folder;req=e.Request;break}}
   a.queue.mu.Unlock()
  }
  if folder==""{fail(w,404,"해당 Triplet Job이 없습니다");return}
  result,err:=tripReadResult(folder,req)
  if err!=nil{fail(w,422,err.Error());return}
  respond(w,200,result)
 case "/api/triplet/compare":
  if r.Method!=http.MethodPost{fail(w,405,"POST만 허용합니다");return}
  var body tripCompareRequest
  if err:=json.NewDecoder(http.MaxBytesReader(w,r.Body,24<<20)).Decode(&body);err!=nil{fail(w,400,"비교 요청 오류");return}
  var folder string;var req RunRequest
  if a.queue!=nil{
   a.queue.mu.Lock()
   for _,e:=range a.queue.Jobs{if e.ID==body.JobID&&e.Job!=nil&&e.Request.Triplet!=nil{folder=e.Job.Folder;req=e.Request;break}}
   a.queue.mu.Unlock()
  }
  if folder==""{fail(w,404,"비교할 Triplet Job이 없습니다");return}
  result,err:=tripReadResult(folder,req)
  if err!=nil{fail(w,422,err.Error());return}
  comparison,err:=tripCompareResult(body.OriginalXYZ,body.OriginalOUT,body.OriginalINP,body.OriginalRoles,result,false)
  if err!=nil{fail(w,422,err.Error());return}
  if body.S0OUT!=""&&body.S0INP!=""{
   scf,e:=ParseOrcaOutput(strings.NewReader(body.S0OUT),"original_S0.out",body.S0INP)
   if e==nil&&scf.Electronic!=nil&&scf.Normal&&scf.SCFConverged&&scf.Multiplicity==1&&scf.Charge==result.Charge&&scf.Method==result.Method&&result.EnergyEh!=nil&&result.Converged{
    delta:=(*result.EnergyEh-*scf.Electronic)*ehToKcal;comparison.ExcitationKcal=&delta
    comparison.Warnings=append(comparison.Warnings,"S₀/T₁ electronic gap (kcal/mol)만 계산했습니다. 서로 다른 최적화 기하에서 측정되었는지, 두 상태가 같은 Method/Target인지 별도 확인하세요.")
   }else{comparison.Warnings=append(comparison.Warnings,"S₀ 에너지 또는 Method/Charge/Spin/수렴 조건이 맞지 않아 Excitation Energy를 표시하지 않습니다")}
  }
  respond(w,200,comparison)
 default:http.NotFound(w,r)
 }
}
var _ io.Reader
