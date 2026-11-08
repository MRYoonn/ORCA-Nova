package main

import (
 "fmt"
 "math"
 "path/filepath"
 "regexp"
 "strconv"
 "strings"
)

type PreflightStage struct {
 Name string `json:"name"`
 Input string `json:"input"`
 Source string `json:"source"`
 Atoms int `json:"atoms,omitempty"`
 Charge int `json:"charge"`
 Multiplicity int `json:"multiplicity"`
 Electrons int `json:"electrons,omitempty"`
}
type PreflightReport struct {
 OK bool `json:"ok"`
 Errors []string `json:"errors"`
 Warnings []string `json:"warnings"`
 Stages []PreflightStage `json:"stages"`
}
var (
 xyzHeader = regexp.MustCompile(`(?im)^\s*\*\s+xyzfile\s+([+-]?\d+)\s+(\d+)\s+(\S+)`)
 palPattern = regexp.MustCompile(`(?im)^\s*nprocs\s+(\d+)`)
 maxcorePattern = regexp.MustCompile(`(?im)^\s*%maxcore\s+(\d+)`)
 atomicNumbers = makeAtomicNumbers()
)
func makeAtomicNumbers() map[string]int {
 symbols:=strings.Fields("H He Li Be B C N O F Ne Na Mg Al Si P S Cl Ar K Ca Sc Ti V Cr Mn Fe Co Ni Cu Zn Ga Ge As Se Br Kr Rb Sr Y Zr Nb Mo Tc Ru Rh Pd Ag Cd In Sn Sb Te I Xe Cs Ba La Ce Pr Nd Pm Sm Eu Gd Tb Dy Ho Er Tm Yb Lu Hf Ta W Re Os Ir Pt Au Hg Tl Pb Bi Po At Rn Fr Ra Ac Th Pa U Np Pu Am Cm Bk Cf Es Fm Md No Lr Rf Db Sg Bh Hs Mt Ds Rg Cn Nh Fl Mc Lv Ts Og")
 result:=make(map[string]int,len(symbols))
 for i,sym:=range symbols { result[sym]=i+1 }
 return result
}
func parseXYZAtoms(data string)(int,int,error){
 lines:=strings.Split(strings.ReplaceAll(data,"\r\n","\n"),"\n")
 if len(lines)<3 { return 0,0,fmt.Errorf("XYZ 원자 수와 주석 행이 필요합니다") }
 count,err:=strconv.Atoi(strings.TrimSpace(lines[0]))
 if err!=nil || count<1 || count>50000 { return 0,0,fmt.Errorf("XYZ 첫 줄의 원자 수가 올바르지 않습니다") }
 if len(lines)<count+2 { return 0,0,fmt.Errorf("XYZ에 %d개 원자의 좌표가 모두 없습니다",count) }
 electrons:=0
 for i:=0;i<count;i++{
  fields:=strings.Fields(lines[i+2])
  if len(fields)<4 { return 0,0,fmt.Errorf("XYZ %d번 원자의 원소와 좌표 3개가 필요합니다",i+1) }
  el:=fields[0]
  if len(el)>2 { return 0,0,fmt.Errorf("XYZ %d번 원소를 인식하지 못했습니다: %s",i+1,el) }
  el=strings.ToUpper(el[:1])+strings.ToLower(el[1:])
  z,ok:=atomicNumbers[el]
  if !ok { return 0,0,fmt.Errorf("XYZ %d번 원소를 인식하지 못했습니다: %s",i+1,fields[0]) }
  electrons+=z
  for _,raw:=range fields[1:4] {
   v,e:=strconv.ParseFloat(strings.ReplaceAll(strings.ReplaceAll(raw,"D","E"),"d","e"),64)
   if e!=nil||math.IsNaN(v)||math.IsInf(v,0){return 0,0,fmt.Errorf("XYZ %d번 원자 좌표는 유한한 숫자여야 합니다",i+1)}
  }
 }
 return count,electrons,nil
}
func preflight(req RunRequest) PreflightReport {
 r:=PreflightReport{OK:true,Errors:[]string{},Warnings:[]string{},Stages:[]PreflightStage{}}
 bad:=func(s string){r.Errors=append(r.Errors,s);r.OK=false}
 if len(req.Files)==0||len(req.Files)>80||len(req.Stages)==0||len(req.Stages)>30{bad("파일은 1–80개, 실행 단계는 1–30개여야 합니다");return r}
 files:=map[string]string{}
 for _,f:=range req.Files{
  ext:=strings.ToLower(filepath.Ext(f.Name));key:=strings.ToLower(f.Name)
  if !safeName(f.Name)||(ext!=".xyz"&&ext!=".inp"){bad("잘못된 파일 이름: "+f.Name);continue}
  if _,ok:=files[key];ok{bad("중복 파일: "+f.Name)}
  if len(f.Content)>4<<20{bad("파일 크기 4MB 초과: "+f.Name)}
  files[key]=f.Content
 }
 type geometry struct{atoms,electrons int}
 geometries:=map[string]geometry{}
 for _,f:=range req.Files {
  if strings.EqualFold(filepath.Ext(f.Name),".xyz"){
   n,z,err:=parseXYZAtoms(f.Content)
   if err!=nil{bad(f.Name+": "+err.Error())}else{geometries[strings.ToLower(f.Name)]=geometry{n,z}}
  }
 }
 for i,st:=range req.Stages {
  prefix:=fmt.Sprintf("%d단계 (%s): ",i+1,st.Name)
  out:=PreflightStage{Name:st.Name,Input:st.Input,Source:st.Source}
  if !safeName(st.Input)||!strings.EqualFold(filepath.Ext(st.Input),".inp"){bad(prefix+"INP 파일명이 잘못됐습니다")}
  inp,hasInput:=files[strings.ToLower(st.Input)]
  if !hasInput{bad(prefix+"INP 누락: "+st.Input)}
  if !safeName(st.Source)||!strings.EqualFold(filepath.Ext(st.Source),".xyz"){bad(prefix+"XYZ 파일명이 잘못됐습니다")}
  matched:=false
  if hasInput {
   m:=xyzHeader.FindStringSubmatch(inp)
   if m==nil{r.Warnings=append(r.Warnings,prefix+"* xyzfile 문장이 보이지 않습니다. 직접 수정한 INP를 검토하세요.")}
   if m!=nil{
    matched=true
    out.Charge,_=strconv.Atoi(m[1]);out.Multiplicity,_=strconv.Atoi(m[2])
    requested:=strings.Trim(m[3],`"'`)
    if !strings.EqualFold(requested,st.Source){bad(prefix+"INP의 XYZ ("+requested+")와 실행 요청 ("+st.Source+")이 다릅니다")}
    if out.Multiplicity<1{bad(prefix+"Multiplicity는 1 이상이어야 합니다")}
   }
   procs:=palPattern.FindStringSubmatch(inp);memory:=maxcorePattern.FindStringSubmatch(inp)
   if procs!=nil&&memory!=nil{
    p,_:=strconv.Atoi(procs[1]);m,_:=strconv.Atoi(memory[1])
    if p<1||m<1{bad(prefix+"nprocs 및 %maxcore는 양수여야 합니다")}
    if int64(p)*int64(m)>60*1024 {r.Warnings=append(r.Warnings,prefix+"nprocs × %maxcore가 60GB를 초과합니다. 실제 RAM을 확인하세요.")}
   }
  }
  g,found:=geometries[strings.ToLower(st.Source)]
  if !found {
   for k:=0;k<i;k++{
    prev:=req.Stages[k];name:=strings.TrimSuffix(prev.Input,filepath.Ext(prev.Input))+".xyz"
    if strings.EqualFold(name,st.Source){
     if !prev.Opt{bad(prefix+"필요한 XYZ를 생성하는 앞 단계가 Optimization이 아닙니다: "+prev.Name)}
     if pg,ok:=geometries[strings.ToLower(prev.Source)];ok{g=pg;found=true}
     break
    }
   }
  }
  if !found{bad(prefix+"XYZ가 없고 앞선 단계에서도 생성되지 않습니다: "+st.Source)}
  if found{
   out.Atoms=g.atoms;out.Electrons=g.electrons-out.Charge
   if out.Electrons<=0{bad(prefix+"Charge 적용 후 전자 수가 0 이하입니다")}
   if matched&&out.Multiplicity>0&&out.Electrons>0 {
    if out.Electrons%2!=(out.Multiplicity-1)%2{bad(prefix+fmt.Sprintf("전자 %d개와 Multiplicity %d의 홀짝성이 맞지 않습니다",out.Electrons,out.Multiplicity))}
    if out.Multiplicity>out.Electrons+1{bad(prefix+"Multiplicity가 전자 수에 비해 너무 큽니다")}
   }
  }
  r.Stages=append(r.Stages,out)
  if st.Opt&&found {geometries[strings.ToLower(strings.TrimSuffix(st.Input,filepath.Ext(st.Input))+".xyz")]=g}
 }
 return r
}
