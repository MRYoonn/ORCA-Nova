package main

import (
 "encoding/json"
 "fmt"
 "math"
 "net/http"
 "regexp"
 "strconv"
 "strings"
)

type SICompositeRequest struct {
 LowOUT string `json:"lowOut"`
 LowINP string `json:"lowInp"`
 LowXYZ string `json:"lowXYZ"`
 HighOUT string `json:"highOut"`
 HighINP string `json:"highInp"`
 HighXYZ string `json:"highXYZ"`
 StationaryPoint string `json:"stationaryPoint"`
}
type SICompositeResult struct {
 Formula string `json:"formula"`
 LowMethod string `json:"lowMethod"`
 HighMethod string `json:"highMethod"`
 Charge int `json:"charge"`
 Multiplicity int `json:"multiplicity"`
 Atoms int `json:"atoms"`
 LowElectronicEh float64 `json:"lowElectronicEh"`
 LowGibbsEh float64 `json:"lowGibbsEh"`
 HighElectronicEh float64 `json:"highElectronicEh"`
 CorrectionEh float64 `json:"correctionEh"`
 CompositeGibbsEh float64 `json:"compositeGibbsEh"`
 Warnings []string `json:"warnings"`
 VerifiedInputs bool `json:"verifiedInputs"`
}
func xyzGeometry(raw string)([]string,[][3]float64,error){
 if len(raw)>4<<20{return nil,nil,fmt.Errorf("XYZ는 4MB를 초과할 수 없습니다")}
 count,_,err:=parseXYZAtoms(raw);if err!=nil{return nil,nil,err}
 if count>400{return nil,nil,fmt.Errorf("기하 구조 비교는 최대 400원자까지 지원합니다")}
 lines:=strings.Split(strings.ReplaceAll(raw,"\r\n","\n"),"\n")
 atoms:=make([]string,0,count);coords:=make([][3]float64,0,count)
 for i:=0;i<count;i++{
  row:=strings.Fields(lines[i+2]);symbol:=strings.ToUpper(row[0][:1])+strings.ToLower(row[0][1:])
  atoms=append(atoms,symbol)
  var xyz [3]float64
  for j:=0;j<3;j++{
   val,e:=strconv.ParseFloat(strings.ReplaceAll(strings.ReplaceAll(row[j+1],"D","E"),"d","e"),64)
   if e!=nil{return nil,nil,e}
   xyz[j]=val
  }
  coords=append(coords,xyz)
 }
 return atoms,coords,nil
}
// Compare all interatomic distances to allow rigid rotation and translation,
// while preventing opt/SP calculations on different conformers or atom order.
func sameXYZGeometry(opt,sp string) (int,error) {
 a,ac,err:=xyzGeometry(opt);if err!=nil{return 0,fmt.Errorf("Opt 최종 XYZ: %w",err)}
 b,bc,err:=xyzGeometry(sp);if err!=nil{return 0,fmt.Errorf("SP 입력 XYZ: %w",err)}
 if len(a)!=len(b) {return 0,fmt.Errorf("Opt/SP XYZ의 원자 수가 다릅니다")}
 for i:=range a{if a[i]!=b[i]{return 0,fmt.Errorf("Opt/SP XYZ의 %d번 원자 종류·순서가 다릅니다",i)}}
 for i:=range ac{
  for j:=0;j<i;j++{
   da:=0.0;db:=0.0
   for k:=0;k<3;k++{da+=(ac[i][k]-ac[j][k])*(ac[i][k]-ac[j][k]);db+=(bc[i][k]-bc[j][k])*(bc[i][k]-bc[j][k])}
   if math.Abs(math.Sqrt(da)-math.Sqrt(db))>0.002{
    return 0,fmt.Errorf("Opt/SP XYZ의 내부 거리 차이가 0.002 Å를 초과합니다 (원자 %d-%d)",i,j)
   }
  }
 }
 return len(a),nil
}
var siSolvKeyword=regexp.MustCompile(`(?i)\b(CPCM|SMD)\s*\(\s*([^()]+)\s*\)`)
func siSolventFromINP(inp string)string{
 m:=siSolvKeyword.FindStringSubmatch(inp)
 if len(m)>2{return strings.ToLower(m[1])+"("+strings.ToLower(strings.Join(strings.Fields(m[2]),""))+")"}
 // ORCA can specify solvent in %cpcm blocks; distinguish unknown from gas.
 if regexp.MustCompile(`(?i)%cpcm\b`).MatchString(inp){return "CPCM(block-unknown)"}
 if regexp.MustCompile(`(?i)%smd\b|\bSMD\b|\bCPCM\b`).MatchString(inp){return "solvation(unspecified)"}
 return "none"
}
func chargeMultiplicity(inp string)(int,int,error){
 m:=chargeMultRx.FindStringSubmatch(inp)
 if len(m)<3{return 0,0,fmt.Errorf("INP의 * xyzfile charge multiplicity를 확인할 수 없습니다")}
 charge,_:=strconv.Atoi(m[1]);mult,_:=strconv.Atoi(m[2])
 return charge,mult,nil
}
func calculateSIComposite(req SICompositeRequest)(SICompositeResult,error){
 res:=SICompositeResult{Formula:"G_composite = E_high(SP) + [G_low(Opt+Freq) - E_low(Opt+Freq)]",Warnings:[]string{}}
 for _,raw:=range []string{req.LowINP,req.HighINP,req.LowOUT,req.HighOUT,req.LowXYZ,req.HighXYZ}{
  if len(raw)==0{return res,fmt.Errorf("Low Opt+Freq 및 High SP의 INP/OUT/XYZ 여섯 파일이 모두 필요합니다")}
  if len(raw)>16<<20{return res,fmt.Errorf("입력 파일당 최대 16MB를 지원합니다")}
 }
 n,err:=sameXYZGeometry(req.LowXYZ,req.HighXYZ)
 if err!=nil{return res,err}
 res.Atoms=n
 q1,m1,err:=chargeMultiplicity(req.LowINP);if err!=nil{return res,err}
 q2,m2,err:=chargeMultiplicity(req.HighINP);if err!=nil{return res,err}
 if q1!=q2||m1!=m2 {return res,fmt.Errorf("Opt/SP의 Charge 또는 Multiplicity가 다릅니다")}
 res.Charge=q1;res.Multiplicity=m1
 s1:=siSolventFromINP(req.LowINP);s2:=siSolventFromINP(req.HighINP)
 if s1!=s2{return res,fmt.Errorf("Opt/SP의 Solvation Model·Solvent 설정이 다릅니다 (%s / %s)",s1,s2)}
 if s1=="CPCM(block-unknown)"||s1=="solvation(unspecified)" {return res,fmt.Errorf("용매가 명시되지 않은 Solvation Model은 혼합수준 Gibbs 자동 결합에서 제외합니다")}
 opt,err:=ParseOrcaOutput(strings.NewReader(req.LowOUT),"low.out",req.LowINP)
 if err!=nil{return res,fmt.Errorf("Opt OUT 파싱 실패: %w",err)}
 sp,err:=ParseOrcaOutput(strings.NewReader(req.HighOUT),"high.out",req.HighINP)
 if err!=nil{return res,fmt.Errorf("SP OUT 파싱 실패: %w",err)}
 if !opt.Normal||!opt.SCFConverged||!opt.Optimized||opt.Electronic==nil||opt.Gibbs==nil {
  return res,fmt.Errorf("Opt+Freq OUT에서 정상종료·SCF·Opt·Electronic·Gibbs가 모두 확인되어야 합니다")
 }
 if !regexp.MustCompile(`(?i)\b(?:Freq|NumFreq|AnFreq)\b`).MatchString(req.LowINP)||opt.ImaginaryCount==nil{
  return res,fmt.Errorf("Opt+Freq OUT의 실제 진동수와 허수 모드 수를 확인할 수 없습니다")
 }
 if !sp.Normal||!sp.SCFConverged||sp.Electronic==nil {
  return res,fmt.Errorf("SP OUT에서 정상종료·SCF·Electronic이 모두 확인되어야 합니다")
 }
 if regexp.MustCompile(`(?i)\bOpt(?:TS)?\b|\bFreq\b`).MatchString(req.HighINP){
  return res,fmt.Errorf("고수준 단계는 구조를 변경하지 않는 Single Point여야 합니다")
 }
 if strings.TrimSpace(req.StationaryPoint)!="minimum" && strings.TrimSpace(req.StationaryPoint)!="ts"{
  return res,fmt.Errorf("Stationary point는 minimum 또는 ts를 지정해야 합니다")
 }
 expected:=0;if req.StationaryPoint=="ts"{expected=1}
 if *opt.ImaginaryCount!=expected{
  return res,fmt.Errorf("%s의 허수 진동수는 %d개여야 합니다 (현재 %d개)",req.StationaryPoint,expected,*opt.ImaginaryCount)
 }
 if opt.Charge!=q1||sp.Charge!=q2||opt.Multiplicity!=m1||sp.Multiplicity!=m2{
  return res,fmt.Errorf("OUT 파싱 결과가 INP Charge/Multiplicity와 일치하지 않습니다")
 }
 for _,energy:=range []float64{*opt.Electronic,*opt.Gibbs,*sp.Electronic}{
  if math.IsNaN(energy)||math.IsInf(energy,0){return res,fmt.Errorf("총 에너지가 유한하지 않습니다")}
 }
 res.LowMethod=opt.Method
 res.HighMethod=sp.Method
 res.LowElectronicEh=*opt.Electronic
 res.LowGibbsEh=*opt.Gibbs
 res.HighElectronicEh=*sp.Electronic
 res.CorrectionEh=*opt.Gibbs-*opt.Electronic
 res.CompositeGibbsEh=*sp.Electronic+res.CorrectionEh
 res.VerifiedInputs=true
 res.Warnings=append(res.Warnings,"이것은 한 화학종의 혼합수준 Gibbs 에너지(Eh)입니다. 반응 ΔG 또는 ΔG‡는 화학양론·표준상태·용매·상태별 비교가 추가로 필요합니다.")
 res.Warnings=append(res.Warnings,"1 atm→1 M 표준상태 보정, 저주파 모드/준조화 보정, 상태별 용매와 basis/correction의 적합성은 자동 적용하지 않습니다.")
 if opt.S2!=nil&&math.Abs(*opt.S2-float64(m1-1)/2*float64(m1+1)/2)>0.1 {
  res.Warnings=append(res.Warnings,"Opt OUT에서 spin contamination 가능성이 있습니다. <S²>를 검토하세요.")
 }
 return res,nil
}
func (a *App) siCompositeRoute(w http.ResponseWriter,r *http.Request){
 if r.Method!=http.MethodPost{fail(w,405,"POST 요청만 허용합니다");return}
 var req SICompositeRequest
 if err:=json.NewDecoder(http.MaxBytesReader(w,r.Body,35<<20)).Decode(&req);err!=nil{
  fail(w,400,"Opt/SP 파일 데이터를 읽지 못했습니다: "+err.Error());return
 }
 res,err:=calculateSIComposite(req)
 if err!=nil{fail(w,422,err.Error());return}
 respond(w,200,res)
}
