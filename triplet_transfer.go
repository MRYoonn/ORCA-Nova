package main

// Triplet Geometry Transfer is geometry preparation, not an electronic-state
// identification algorithm. The atom labels C2/C5/C6 are chemical ring roles,
// NEVER raw XYZ row indices.
import (
 "encoding/json"
 "fmt"
 "math"
 "net/http"
 "io"
 "sort"
 "strings"
)

type tripVec [3]float64
type tripMolecule struct { Atoms []string; XYZ []tripVec; Bond [][]bool }
type tripRoles map[string]int
var tripRingRoles=[]string{"C2","N3","C4","C5","C6","N1"}
var tripExtraRoles=[]string{"S2","O4","H1","H3","H5","H6"}
var tripRadii=map[string]float64{"H":0.31,"C":0.76,"N":0.71,"O":0.66,"F":0.57,"S":1.05,"Cl":1.02,"Br":1.20,"P":1.07,"B":0.84,"Si":1.11}
func vecAdd(a,b tripVec)tripVec{return tripVec{a[0]+b[0],a[1]+b[1],a[2]+b[2]}}
func vecSub(a,b tripVec)tripVec{return tripVec{a[0]-b[0],a[1]-b[1],a[2]-b[2]}}
func vecScale(a tripVec,x float64)tripVec{return tripVec{a[0]*x,a[1]*x,a[2]*x}}
func vecDot(a,b tripVec)float64{return a[0]*b[0]+a[1]*b[1]+a[2]*b[2]}
func vecNorm(a tripVec)float64{return math.Sqrt(vecDot(a,a))}
func vecUnit(a tripVec)(tripVec,error){v:=vecNorm(a);if v<1e-6{return tripVec{},fmt.Errorf("선택 원자 좌표가 겹쳐 정렬할 수 없습니다")};return vecScale(a,1/v),nil}
func vecCross(a,b tripVec)tripVec{return tripVec{a[1]*b[2]-a[2]*b[1],a[2]*b[0]-a[0]*b[2],a[0]*b[1]-a[1]*b[0]}}
func tripDistance(a,b tripVec)float64{return vecNorm(vecSub(a,b))}
func parseTripletXYZ(raw string)(tripMolecule,error){
 var m tripMolecule
 atoms,coords,err:=xyzGeometry(raw)
 if err!=nil{return m,err}
 if len(atoms)>240 {return m,fmt.Errorf("Triplet Geometry Transfer는 최대 240원자까지 지원합니다")}
 for i,a:=range atoms{
  if _,ok:=tripRadii[a];!ok{return m,fmt.Errorf("결합 그래프를 지원하지 않는 원소: %s (index %d)",a,i)}
  m.Atoms=append(m.Atoms,a)
  m.XYZ=append(m.XYZ,tripVec(coords[i]))
 }
 n:=len(atoms);m.Bond=make([][]bool,n)
 for i:=range m.Bond{m.Bond[i]=make([]bool,n)}
 for i:=0;i<n;i++{for j:=0;j<i;j++{
  if atoms[i]=="H"&&atoms[j]=="H"{continue}
  distance:=tripDistance(m.XYZ[i],m.XYZ[j])
  // Conservative covalent-radius bond inference. Weak interactions, metal
  // complexes and ambiguous heavy-element connectivity require manual review.
  threshold:=1.22*(tripRadii[atoms[i]]+tripRadii[atoms[j]])
  if distance>=0.43&&distance<=threshold{m.Bond[i][j]=true;m.Bond[j][i]=true}
 }}
 return m,nil
}
func tripNeighbors(m tripMolecule,i int,symbol string)[]int{
 out:=[]int{}
 if i<0||i>=len(m.Atoms){return out}
 for j,b:=range m.Bond[i]{if b&&(symbol==""||m.Atoms[j]==symbol){out=append(out,j)}}
 return out
}
func tripLinked(m tripMolecule,i,j int)bool{return i>=0&&j>=0&&i<len(m.Atoms)&&j<len(m.Atoms)&&m.Bond[i][j]}
func tripExternalNeighbor(m tripMolecule,idx int,symbol string,exclude map[int]bool)[]int{
 out:=[]int{}
 for _,j:=range tripNeighbors(m,idx,symbol){if !exclude[j]{out=append(out,j)}}
 return out
}
func tripTryRoles(m tripMolecule)([]tripRoles,error){
 unique:=map[string]tripRoles{}
 for c2,element:=range m.Atoms {
  if element!="C"{continue}
  if len(tripNeighbors(m,c2,"N"))!=2 || len(tripNeighbors(m,c2,"S"))!=1{continue}
  for _,n3:=range tripNeighbors(m,c2,"N"){
   for _,c4:=range tripNeighbors(m,n3,"C"){
    if c4==c2||len(tripNeighbors(m,c4,"O"))!=1{continue}
    for _,c5:=range tripNeighbors(m,c4,"C"){
     if c5==c2{continue}
     for _,c6:=range tripNeighbors(m,c5,"C"){
      if c6==c4{continue}
      for _,n1:=range tripNeighbors(m,c6,"N"){
       if n1==n3||!tripLinked(m,n1,c2){continue}
       ring:=[]int{c2,n3,c4,c5,c6,n1}
       seen:=map[int]bool{};valid:=true
       for _,v:=range ring{if seen[v]{valid=false};seen[v]=true}
       if !valid {continue}
       // Both ring nitrogens and the carbonyl/carbon-sulfur anchors must
       // produce the pyrimidine graph. Reject ambiguous fused rings.
       for i:=0;i<len(ring);i++{
        nRing:=0
        for _,x:=range ring{if tripLinked(m,ring[i],x){nRing++}}
        if nRing!=2{valid=false}
       }
       if !valid{continue}
       sulfurs:=tripExternalNeighbor(m,c2,"S",seen)
       oxygens:=tripExternalNeighbor(m,c4,"O",seen)
       if len(sulfurs)!=1||len(oxygens)!=1{continue}
       roles:=tripRoles{"C2":c2,"N3":n3,"C4":c4,"C5":c5,"C6":c6,"N1":n1,"S2":sulfurs[0],"O4":oxygens[0]}
       for role,atom:=range map[string]int{"H1":n1,"H3":n3,"H5":c5,"H6":c6}{
        h:=tripNeighbors(m,atom,"H");if len(h)==1{roles[role]=h[0]}
       }
       key:=fmt.Sprint(ring)
       unique[key]=roles
      }
     }
    }
   }
  }
 }
 keys:=make([]string,0,len(unique));for k:=range unique{keys=append(keys,k)};sort.Strings(keys)
 result:=[]tripRoles{};for _,key:=range keys{result=append(result,unique[key])}
 if len(result)==0{return nil,fmt.Errorf("명확한 thiouracil C2(S)-N3-C4(O)-C5-C6-N1 고리 그래프가 아닙니다. XYZ 결합·protonation을 확인하세요")}
 return result,nil
}
func tripCheckRoles(m tripMolecule,r tripRoles)error{
 if len(r)<8 {return fmt.Errorf("C2/N3/C4/C5/C6/N1/S2/O4 원자 매핑 8개가 필요합니다")}
 expected:=map[string]string{"C2":"C","N3":"N","C4":"C","C5":"C","C6":"C","N1":"N","S2":"S","O4":"O"}
 seen:=map[int]bool{}
 for role,kind:=range expected{
  i,ok:=r[role];if !ok||i<0||i>=len(m.Atoms){return fmt.Errorf("%s의 0-based 원자 index가 유효하지 않습니다",role)}
  if m.Atoms[i]!=kind{return fmt.Errorf("%s: 기대 원소 %s, 실제 %s",role,kind,m.Atoms[i])}
  if seen[i]{return fmt.Errorf("중복 원자 매핑 (index %d)",i)};seen[i]=true
 }
 cycle:=[]string{"C2","N3","C4","C5","C6","N1"}
 for k:=0;k<len(cycle);k++{if !tripLinked(m,r[cycle[k]],r[cycle[(k+1)%len(cycle)]]){return fmt.Errorf("%s-%s 결합이 XYZ에서 식별되지 않습니다",cycle[k],cycle[(k+1)%len(cycle)])}}
 if !tripLinked(m,r["C2"],r["S2"])||!tripLinked(m,r["C4"],r["O4"]){return fmt.Errorf("C2–S / C4–O 접속이 일치하지 않습니다")}
 for label,parent:=range map[string]string{"H1":"N1","H3":"N3","H5":"C5","H6":"C6"}{
  if index,ok:=r[label];ok{
   if index<0||index>=len(m.Atoms)||m.Atoms[index]!="H"||!tripLinked(m,index,r[parent]){return fmt.Errorf("%s의 고리 H 연결성이 다릅니다",label)}
  }
 }
 // Fused rings or missing attachment might change the role meaning.
 for _,key:=range cycle{
  num:=0;for _,role:=range cycle{if tripLinked(m,r[key],r[role]){num++}}
  if num!=2{return fmt.Errorf("고리 역할 %s의 추가 고리 결합이 있어 모호합니다",key)}
 }
 return nil
}
func tripRoleHydrogens(m tripMolecule,r tripRoles,role string)int{return len(tripNeighbors(m,r[role],"H"))}
func tripValidatePair(ref,target tripMolecule,rr,tr tripRoles,refCharge,targetCharge int)error{
 if err:=tripCheckRoles(ref,rr);err!=nil{return fmt.Errorf("Reference: %w",err)}
 if err:=tripCheckRoles(target,tr);err!=nil{return fmt.Errorf("Target: %w",err)}
 if refCharge!=targetCharge{return fmt.Errorf("Reference와 Target의 선언된 Charge가 다릅니다. protonation과 전자배치를 확인하세요")}
 for _,role:=range []string{"N1","N3","C5","C6"}{
  a,b:=tripRoleHydrogens(ref,rr,role),tripRoleHydrogens(target,tr,role)
  if a!=b{return fmt.Errorf("%s에 결합된 H 수가 서로 다릅니다 (Reference=%d, Target=%d). protonation 위치가 다르면 이식 금지",role,a,b)}
 }
 return nil
}
type tripFrame struct{origin,x,y,z tripVec}
func makeTripFrame(m tripMolecule,r tripRoles)(tripFrame,error){
 var f tripFrame
 f.origin=m.XYZ[r["C2"]]
 var err error
 f.x,err=vecUnit(vecSub(m.XYZ[r["C4"]],f.origin));if err!=nil{return f,err}
 second:=vecSub(m.XYZ[r["C6"]],f.origin)
 f.y,err=vecUnit(vecSub(second,vecScale(f.x,vecDot(second,f.x))));if err!=nil{return f,err}
 f.z=vecCross(f.x,f.y)
 return f,nil
}
func tripTransform(p tripVec,source,dest tripFrame)tripVec{
 u:=vecSub(p,source.origin)
 return vecAdd(dest.origin,vecAdd(vecAdd(vecScale(dest.x,vecDot(u,source.x)),vecScale(dest.y,vecDot(u,source.y))),vecScale(dest.z,vecDot(u,source.z))))
}
func tripXYZText(m tripMolecule,label string)string{
 var b strings.Builder
 fmt.Fprintf(&b,"%d\n%s\n",len(m.Atoms),label)
 for i,a:=range m.Atoms{v:=m.XYZ[i];fmt.Fprintf(&b,"%-2s % .10f % .10f % .10f\n",a,v[0],v[1],v[2])}
 return b.String()
}
func tripTransfer(ref,target tripMolecule,rr,tr tripRoles,selected []string)(tripMolecule,[]string,error){
 out:=target
 out.XYZ=append([]tripVec(nil),target.XYZ...)
 af,err:=makeTripFrame(target,tr);if err!=nil{return out,nil,err}
 bf,err:=makeTripFrame(ref,rr);if err!=nil{return out,nil,err}
 for i,p:=range out.XYZ{out.XYZ[i]=tripTransform(p,af,bf)}
 for _,role:=range selected{
  a,okA:=rr[role];b,okB:=tr[role]
  if !okA||!okB{return out,nil,fmt.Errorf("%s의 Reference/Target 매핑이 없습니다",role)}
  if ref.Atoms[a]!=target.Atoms[b]{return out,nil,fmt.Errorf("%s 매핑 원소가 일치하지 않습니다",role)}
  out.XYZ[b]=ref.XYZ[a]
 }
 warnings:=[]string{}
 // Use original TARGET connectivity as ground truth, not the ref. Bond
 // inference is only a safety heuristic; serious geometry changes are flagged.
 for i:=range target.Atoms{for j:=0;j<i;j++{
  d:=tripDistance(out.XYZ[i],out.XYZ[j])
  if target.Bond[i][j]{
   old:=tripDistance(target.XYZ[i],target.XYZ[j])
   if d<0.55||d>old*1.35||d<old*0.70{
    warnings=append(warnings,fmt.Sprintf("Target bond %d-%d (%s-%s) length %.3f Å, original %.3f Å: 검토 필요",j,i,target.Atoms[j],target.Atoms[i],d,old))
   }
  }else if d<0.65{
   warnings=append(warnings,fmt.Sprintf("새 좌표에 원자 충돌 가능성: %d-%d %.3f Å",j,i,d))
  }
 }}
 if len(warnings)>30{warnings=append(warnings[:30],"추가 충돌·결합 경고가 30개를 초과했습니다")}
 return out,warnings,nil
}
func tripRoleArray(r tripRoles)[]string{
 out:=[]string{}
 for _,role:=range append(append([]string{},tripRingRoles...),tripExtraRoles...){out=append(out,fmt.Sprintf("%s=%d",role,r[role]))}
 return out
}
func tripRoleDistance(m tripMolecule,r tripRoles,a,b string)float64{return tripDistance(m.XYZ[r[a]],m.XYZ[r[b]])}
func tripCovalentSafeName(name string)string{
 name=strings.TrimSuffix(name,".xyz")
 return safeLabel(name)
}

// Shared read-only analyzer and strict geometry transfer. A submitted job must
// be prepared again using the same reviewed mapping, never raw index guessing.
type tripTargetRequest struct{ Name string `json:"name"`; XYZ string `json:"xyz"`; Roles tripRoles `json:"roles,omitempty"` }
type tripOptions struct{
 Charge int `json:"charge"`
 ReferenceCharge int `json:"referenceCharge"`
 Multiplicity int `json:"multiplicity"`
 Constrained bool `json:"constrained"`
 Unconstrained bool `json:"unconstrained"`
 Spin bool `json:"spin"`
 Frequency bool `json:"frequency"`
 FixedRoles []string `json:"fixedRoles"`
 Functional string `json:"functional"`
 Basis string `json:"basis"`
 Dispersion string `json:"dispersion"`
 Solvent string `json:"solvent"`
 Cores int `json:"cores"`
 MaxCore int `json:"maxcore"`
}
type tripJobMeta struct{
 TargetName string `json:"targetName"`
 Roles tripRoles `json:"roles"`
 ReferenceRoles tripRoles `json:"referenceRoles"`
 Charge int `json:"charge"`
 Multiplicity int `json:"multiplicity"`
 Frequency bool `json:"frequency"`
 SelectedRoles []string `json:"selectedRoles"`
 Confirmed bool `json:"confirmed"`
}
type tripPrepareRequest struct{
 Reference string `json:"reference"`
 ReferenceRoles tripRoles `json:"referenceRoles,omitempty"`
 Targets []tripTargetRequest `json:"targets"`
 Options tripOptions `json:"options"`
 Confirmed bool `json:"confirmed"`
}
type tripPrepared struct{
 Name string `json:"name"`
 ReferenceRoles tripRoles `json:"referenceRoles"`
 TargetRoles tripRoles `json:"targetRoles"`
 TransferredXYZ string `json:"transferredXYZ"`
 Warnings []string `json:"warnings"`
 Request *RunRequest `json:"request,omitempty"`
}
type tripPrepareResponse struct{Items []tripPrepared `json:"items"`;Warnings []string `json:"warnings"`;NeedsConfirmation bool `json:"needsConfirmation"`}
func (o tripOptions) validate()error{
 if o.Multiplicity!=3{return fmt.Errorf("UKS Triplet는 Multiplicity 3이어야 합니다")}
 if o.Charge< -3||o.Charge>3{return fmt.Errorf("Charge 범위는 -3~+3입니다")}
 if !o.Unconstrained{return fmt.Errorf("전체 Triplet 최적화(STEP 2)는 반드시 활성화해야 합니다")}
 if !o.Constrained{return fmt.Errorf("Geometry Transfer에는 먼저 고리 Constraints 최적화가 필요합니다")}
 if !o.Spin{return fmt.Errorf("Spin Density 분석(STEP 3)을 활성화해야 합니다")}
 if o.Cores<1||o.Cores>64||o.MaxCore<100||o.MaxCore>16000{return fmt.Errorf("CPU 1–64 및 MaxCore 100–16000 MB 범위를 지정하세요")}
 if o.Functional!="B3LYP"&&o.Functional!="PBE0"&&o.Functional!="M06-2X"{return fmt.Errorf("지원되지 않는 Functional (B3LYP/PBE0/M06-2X)")}
 if o.Basis!="def2-TZVP"&&o.Basis!="def2-SVP"&&o.Basis!="def2-TZVPP"{return fmt.Errorf("지원되지 않는 Basis")}
 if o.Dispersion!="D4"&&o.Dispersion!="D3BJ"{return fmt.Errorf("지원되지 않는 Dispersion")}
 if o.Solvent!="acetonitrile"&&o.Solvent!="toluene"&&o.Solvent!="water"{return fmt.Errorf("지원되지 않는 SMD 용매")}
 if len(o.FixedRoles)==0{return fmt.Errorf("고정할 고리 원자를 1개 이상 선택하세요")}
 seen:=map[string]bool{}
 for _,v:=range o.FixedRoles{
  if seen[v]{return fmt.Errorf("중복 Constraints: %s",v)};seen[v]=true
  valid:=false;for _,role:=range append(append([]string{},tripRingRoles...),tripExtraRoles...){if v==role{valid=true}}
  if !valid{return fmt.Errorf("고정할 수 없는 원자 Role: %s",v)}
 }
 return nil
}
func tripORCAInput(stage,source string,opt tripOptions,freeze []int)string{
 task:="Opt";if stage=="spin"{task="SP"}else if stage=="free"&&opt.Frequency{task="Opt Freq"}
 var b strings.Builder
 fmt.Fprintf(&b,"! UKS %s %s %s TightSCF %s SMD(%s)\n",opt.Functional,opt.Dispersion,opt.Basis,task,opt.Solvent)
 fmt.Fprintf(&b,"%%pal\n nprocs %d\nend\n\n%%maxcore %d\n",opt.Cores,opt.MaxCore)
 if len(freeze)>0{
  b.WriteString("\n%geom\n Constraints\n")
  for _,i:=range freeze{fmt.Fprintf(&b,"  { C %d C }\n",i)}
  b.WriteString(" end\nend\n")
 }
 if stage=="spin"{
  b.WriteString("\n%output\n Print[ P_Mulliken ] 1\n Print[ P_Loewdin ] 1\n Print[ P_Hirshfeld ] 1\nend\n")
  b.WriteString("\n%plots\n Format Gaussian_Cube\n dim1 64\n dim2 64\n dim3 64\n SpinDens(\"step3_spin.cube\");\nend\n")
 }
 fmt.Fprintf(&b,"\n* xyzfile %d 3 %s\n",opt.Charge,source)
 return b.String()
}
func tripBuildRequest(name,initial string,target tripMolecule,role tripRoles,opts tripOptions)RunRequest{
 indices:=[]int{}
 for _,r:=range opts.FixedRoles{indices=append(indices,role[r])};sort.Ints(indices)
 first:=tripORCAInput("fixed","transferred_initial.xyz",opts,indices)
 second:=tripORCAInput("free","step1_constrained.xyz",opts,nil)
 third:=tripORCAInput("spin","step2_triplet.xyz",opts,nil)
 return RunRequest{Label:"TGT_"+tripCovalentSafeName(name),Triplet:&tripJobMeta{TargetName:name,Roles:role,Charge:opts.Charge,Multiplicity:3,Frequency:opts.Frequency,SelectedRoles:append([]string(nil),opts.FixedRoles...),Confirmed:true},Files:[]JobFile{
  {Name:"transferred_initial.xyz",Content:initial},
  {Name:"step1_constrained.inp",Content:first},
  {Name:"step2_triplet.inp",Content:second},
  {Name:"step3_spin.inp",Content:third},
 },Stages:[]Stage{
  {Name:"Constrained Triplet Opt",Input:"step1_constrained.inp",Source:"transferred_initial.xyz",Opt:true},
  {Name:"Unconstrained Triplet Opt",Input:"step2_triplet.inp",Source:"step1_constrained.xyz",Opt:true},
  {Name:"Triplet Spin Density",Input:"step3_spin.inp",Source:"step2_triplet.xyz",Opt:false},
 }}
}
func prepareTriplet(req tripPrepareRequest)(tripPrepareResponse,error){
 response:=tripPrepareResponse{Items:[]tripPrepared{},NeedsConfirmation:!req.Confirmed,Warnings:[]string{}}
 if err:=req.Options.validate();err!=nil{return response,err}
 if len(req.Targets)==0||len(req.Targets)>24{return response,fmt.Errorf("한 번에 Target XYZ 1~24개를 불러올 수 있습니다")}
 reference,err:=parseTripletXYZ(req.Reference);if err!=nil{return response,fmt.Errorf("Reference XYZ: %w",err)}
 candidates,err:=tripTryRoles(reference);if err!=nil{return response,err}
 rr:=req.ReferenceRoles
 if len(rr)==0{
  if len(candidates)!=1{return response,fmt.Errorf("Reference 고리 후보가 %d개로 모호합니다. 원자 Role 매핑을 직접 지정하세요",len(candidates))}
  rr=candidates[0]
 }
 if err:=tripCheckRoles(reference,rr);err!=nil{return response,err}
 for _,role:=range req.Options.FixedRoles{
  if _,ok:=rr[role];!ok{return response,fmt.Errorf("Reference에는 선택한 %s 원자가 없습니다",role)}
 }
 response.Warnings=append(response.Warnings,"원자 연결성은 XYZ 좌표의 공유결합 반지름으로 추정했습니다. 전하, N1 protonation, Thiouracil tautomer, UKS Triplet state identity는 사람이 검증해야 합니다.")
 for _,entry:=range req.Targets{
  if !safeName(entry.Name)||!strings.HasSuffix(strings.ToLower(entry.Name),".xyz"){return response,fmt.Errorf("안전한 Target XYZ 파일명이 필요합니다: %s",entry.Name)}
  target,err:=parseTripletXYZ(entry.XYZ);if err!=nil{return response,fmt.Errorf("%s: %w",entry.Name,err)}
  candidate,err:=tripTryRoles(target);if err!=nil{return response,fmt.Errorf("%s: %w",entry.Name,err)}
  tr:=entry.Roles
  if len(tr)==0{
   if len(candidate)!=1{return response,fmt.Errorf("%s: Target 고리 후보가 %d개입니다. Role을 직접 지정하세요",entry.Name,len(candidate))}
   tr=candidate[0]
  }
  if err:=tripValidatePair(reference,target,rr,tr,req.Options.ReferenceCharge,req.Options.Charge);err!=nil{return response,fmt.Errorf("%s: %w",entry.Name,err)}
  for _,role:=range req.Options.FixedRoles{if _,ok:=tr[role];!ok{return response,fmt.Errorf("%s: Target에 선택한 %s 원자가 없습니다",entry.Name,role)} }
  aligned,warn,err:=tripTransfer(reference,target,rr,tr,req.Options.FixedRoles)
  if err!=nil{return response,fmt.Errorf("%s: %w",entry.Name,err)}
  initial:=tripXYZText(aligned,"Transfer Target="+entry.Name+"; original Target identity/order preserved; Mapping confirmed="+fmt.Sprint(req.Confirmed))
  p:=tripPrepared{Name:entry.Name,ReferenceRoles:rr,TargetRoles:tr,TransferredXYZ:initial,Warnings:warn}
  if req.Confirmed{
   run:=tripBuildRequest(entry.Name,initial,target,tr,req.Options)
   run.Triplet.ReferenceRoles=rr
   pf:=preflight(run);if !pf.OK{return response,fmt.Errorf("%s: 계산 전 검사 실패: %s",entry.Name,strings.Join(pf.Errors,"; "))}
   p.Request=&run
  }
  response.Items=append(response.Items,p)
 }
 return response,nil
}
func (a *App) tripletRoute(w http.ResponseWriter,r *http.Request){
 if r.Method!=http.MethodPost{fail(w,405,"POST만 허용합니다");return}
 switch r.URL.Path{
 case "/api/triplet/prepare":
  var req tripPrepareRequest
  if err:=json.NewDecoder(http.MaxBytesReader(w,r.Body,70<<20)).Decode(&req);err!=nil{fail(w,400,"XYZ 요청을 읽을 수 없습니다: "+err.Error());return}
  res,err:=prepareTriplet(req)
  if err!=nil{fail(w,422,err.Error());return}
  respond(w,200,res)
 default:a.tripletExtraRoute(w,r)
 }
}
var _ io.Reader
