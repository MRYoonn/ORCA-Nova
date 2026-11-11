package main

import(
 "fmt"
 "math"
 "strings"
 "testing"
)

func exampleTripletXYZ(ethyl bool, reorder bool, protonateN1 bool) string{
 atoms:=[]struct{e string;x,y,z float64}{
 {"C",1.400,0,0},{"N",0.700,1.212,0},{"C",-0.700,1.212,0},{"C",-1.400,0,0},{"C",-0.700,-1.212,0},{"N",0.700,-1.212,0},
 {"S",3.180,0,0},{"O",-1.325,2.310,0},
 {"H",1.240,-2.120,0},{"H",1.240,2.120,0},{"H",-2.500,0,0},{"H",-1.250,-2.160,0},
 {"C",4.890,0,0},
 }
 if !protonateN1{atoms[8].x=1.780;atoms[8].y=2.040} // H moves to N3; mismatched protonation
 if ethyl {
  atoms=append(atoms,
   struct{e string;x,y,z float64}{"C",6.410,0,0},
   struct{e string;x,y,z float64}{"H",5.260,0.55,0.85},
   struct{e string;x,y,z float64}{"H",5.260,0.55,-0.85},
   struct{e string;x,y,z float64}{"H",6.800,0.89,0.48},
   struct{e string;x,y,z float64}{"H",6.800,-0.89,0.48},
   struct{e string;x,y,z float64}{"H",6.800,0,-1.03},
  )
 }else {
  atoms=append(atoms,
   struct{e string;x,y,z float64}{"H",5.250,0.98,0},
   struct{e string;x,y,z float64}{"H",5.250,-0.49,0.85},
   struct{e string;x,y,z float64}{"H",5.250,-0.49,-0.85},
  )
 }
 if reorder{
  // Different XYZ atom ordering plus rotation+translation; chemistry unchanged.
  reordered:=append([]struct{e string;x,y,z float64}{},atoms[6:]...)
  reordered=append(reordered,atoms[:6]...)
  atoms=reordered
 }
 var b strings.Builder
 fmt.Fprintf(&b,"%d\nsame thiouracil scaffold\n",len(atoms))
 for _,a:=range atoms{
  x,y,z:=a.x,a.y,a.z
  if reorder{x,y,z=-y+8,x-2,z+3}
  fmt.Fprintf(&b,"%s %.7f %.7f %.7f\n",a.e,x,y,z)
 }
 return b.String()
}
func standardTripOpts()tripOptions{return tripOptions{
 Charge:1,ReferenceCharge:1,Multiplicity:3,Constrained:true,Unconstrained:true,Spin:true,
 FixedRoles:append([]string{},tripRingRoles...),Functional:"B3LYP",Basis:"def2-TZVP",Dispersion:"D4",
 Solvent:"acetonitrile",Cores:8,MaxCore:2500,
}}
func TestTripletRingMappingDifferentXYZAtomOrder(t *testing.T){
 ref,err:=parseTripletXYZ(exampleTripletXYZ(false,false,true));if err!=nil{t.Fatal(err)}
 target,err:=parseTripletXYZ(exampleTripletXYZ(true,true,true));if err!=nil{t.Fatal(err)}
 r,err:=tripTryRoles(ref);if err!=nil{t.Fatal(err)}
 g,err:=tripTryRoles(target);if err!=nil{t.Fatal(err)}
 if len(r)!=1||len(g)!=1{t.Fatalf("ambiguous roles: %#v %#v",r,g)}
 if r[0]["C2"]!=0||g[0]["C2"]==0 {t.Fatal("Atom mapping accidentally assumed identical row indices",r,g)}
 if err:=tripValidatePair(ref,target,r[0],g[0],1,1);err!=nil{t.Fatal(err)}
 out,warn,err:=tripTransfer(ref,target,r[0],g[0],tripRingRoles);if err!=nil{t.Fatal(err)}
 for _,role:=range tripRingRoles{
  if tripDistance(out.XYZ[g[0][role]],ref.XYZ[r[0][role]])>1e-7{t.Fatalf("mapped role %s not transferred exactly",role)}
 }
 if out.Atoms[g[0]["S2"]]!="S"||out.Atoms[g[0]["C2"]]!="C"{t.Fatal("Target atom composition changed")}
 if len(out.Atoms)!=len(target.Atoms) {t.Fatal("Target sidechain was lost")}
 if len(warn)>30{t.Fatal("unexpected warning overflow")}
}
func TestTripletStagesConstrainedAndUnconstrained(t *testing.T){
 req:=tripPrepareRequest{
  Reference:exampleTripletXYZ(false,false,true),
  Targets:[]tripTargetRequest{{Name:"NH_S-nPr.xyz",XYZ:exampleTripletXYZ(true,true,true)}},
  Options:standardTripOpts(),
 }
 draft,err:=prepareTriplet(req);if err!=nil{t.Fatal(err)}
 if !draft.NeedsConfirmation||draft.Items[0].Request!=nil{t.Fatal("unconfirmed mapping was made executable")}
 req.Confirmed=true
 result,err:=prepareTriplet(req);if err!=nil{t.Fatal(err)}
 job:=result.Items[0].Request
 if job==nil||len(job.Stages)!=3||len(job.Files)!=4{t.Fatal("invalid 3-stage transfer job",job)}
 first,second,third:=job.Files[1].Content,job.Files[2].Content,job.Files[3].Content
 if !strings.Contains(first,"%geom\n Constraints\n")||strings.Count(first,"{ C ")!=6{t.Fatal("ring atom Cartesian constraints wrong",first)}
 if strings.Contains(second,"%geom")||strings.Contains(second,"Constraints"){t.Fatal("stage2 must not constrain ring",second)}
 if job.Stages[1].Source!="step1_constrained.xyz"||job.Stages[2].Source!="step2_triplet.xyz"{t.Fatal("optimized geometry was not chained")}
 if !strings.Contains(third,"Print[ P_Hirshfeld ]")||!strings.Contains(third,"SpinDens(\"step3_spin.cube\")"){t.Fatal("Spin Density/Populations not requested")}
 if strings.Contains(third,"%geom"){t.Fatal("stage3 should use fully relaxed geometry")}
 if !strings.Contains(third,"MOREAD")||!strings.Contains(third,`%moinp "step2_triplet.gbw"`){t.Fatal("Step3 must reuse converged UKS orbitals from Step2 GBW")}
 report:=preflight(*job);if !report.OK{t.Fatal(report.Errors)}
 cpu,mem,err:=resources(*job);if err!=nil||cpu!=8||mem<20000{t.Fatalf("scheduler reservation invalid %d %d %v",cpu,mem,err)}
}
func TestTripletProtonationAndChargeMismatchRefused(t *testing.T){
 base:=tripPrepareRequest{Reference:exampleTripletXYZ(false,false,true),
  Targets:[]tripTargetRequest{{Name:"target.xyz",XYZ:exampleTripletXYZ(true,true,false)}},
  Options:standardTripOpts(),Confirmed:true,
 }
 if _,err:=prepareTriplet(base);err==nil{t.Fatal("protonation change was not rejected")}
 base.Targets[0].XYZ=exampleTripletXYZ(true,true,true)
 base.Options.ReferenceCharge=0
 if _,err:=prepareTriplet(base);err==nil{t.Fatal("different charge was not rejected")}
 base.Options.ReferenceCharge=1
 base.Options.Charge=0
 if _,err:=prepareTriplet(base);err==nil{t.Fatal("different charge was not rejected")}
 base.Options.Charge=1
 base.Targets[0].Roles=tripRoles{"C2":0,"N3":1,"C4":2,"C5":3,"C6":4,"N1":5,"S2":6,"O4":7}
 if _,err:=prepareTriplet(base);err==nil{t.Fatal("user-supplied incorrect index mapping was accepted")}
}
func TestTripletRingDistancesAndClashWarnings(t *testing.T){
 ref,_:=parseTripletXYZ(exampleTripletXYZ(false,false,true))
 target,_:=parseTripletXYZ(exampleTripletXYZ(true,false,true))
 rr,_:=tripTryRoles(ref);tr,_:=tripTryRoles(target)
 // An extra selected S coordinate must transfer from the reference and
 // warn if original attachment geometry is changed.
 sel:=append(append([]string{},tripRingRoles...),"S2")
 out,_,err:=tripTransfer(ref,target,rr[0],tr[0],sel)
 if err!=nil{t.Fatal(err)}
 if math.Abs(tripRoleDistance(out,tr[0],"C2","S2")-tripRoleDistance(ref,rr[0],"C2","S2"))>1e-8{t.Fatal("S2 coordinate transfer failed")}
}
func TestTripletNoImplicitScientificStateConclusion(t *testing.T){
 req:=tripPrepareRequest{Reference:exampleTripletXYZ(false,false,true),Targets:[]tripTargetRequest{{Name:"S-Me.xyz",XYZ:exampleTripletXYZ(false,false,true)}},Options:standardTripOpts(),Confirmed:true}
 res,err:=prepareTriplet(req);if err!=nil{t.Fatal(err)}
 if len(res.Warnings)==0||!strings.Contains(strings.Join(res.Warnings,""),"UKS Triplet state identity"){t.Fatal("must disclose unverified triplet electronic state")}
}

func TestTripletInternalGBWReuseNeverAllowsExternalOrbitalInjection(t *testing.T){
 opts:=standardTripOpts();opts.Cores=2;opts.MaxCore=200
 req:=tripPrepareRequest{Reference:exampleTripletXYZ(false,false,true),Options:opts,
  Targets:[]tripTargetRequest{{Name:"test.xyz",XYZ:exampleTripletXYZ(true,false,true)}},Confirmed:true}
 result,err:=prepareTriplet(req);if err!=nil{t.Fatal(err)}
 job:=*result.Items[0].Request
 if !tripAllowInternalMORead(job,job.Stages[2],job.Files[3].Content){t.Fatal("safe internal STEP2 .gbw rejected")}
 if _,_,err:=resources(job);err!=nil{t.Fatal("safe internal STEP2 .gbw broke scheduler",err)}
 malicious:=[]string{
   strings.Replace(job.Files[3].Content,"step2_triplet.gbw","..\\other_molecule.gbw",1),
   job.Files[3].Content+"\n%moinp \"external.gbw\"\n",
   job.Files[3].Content+"\n$new_job\n",
   strings.Replace(job.Files[3].Content,"step2_triplet.gbw","step3_spin.gbw",1),
 }
 for i,content:=range malicious{
  copyJob:=job
  copyJob.Files=append([]JobFile(nil),job.Files...)
  copyJob.Files[3].Content=content
  if _,_,err:=resources(copyJob);err==nil{t.Errorf("malicious MORead input #%d accepted",i)}
 }
 // No global exceptions: ordinary Batch calculations cannot access orbitals.
 copyJob:=job;copyJob.Triplet=nil
 if _,_,err:=resources(copyJob);err==nil{t.Fatal("ordinary Batch allowed MOREAD")}
}
