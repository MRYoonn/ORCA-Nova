package main

import(
 "fmt"
 "math"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestTripletMockTwoTargetBatchStagesAndScientificOutputs(t *testing.T){
 t.Setenv("ORCA_NOVA_TRIPLET_MOCK","1")
 queue:=mockQueue(t)
 q:=queue
 options:=standardTripOpts();options.Cores=2;options.MaxCore=200
 options.Frequency=false
 req:=tripPrepareRequest{Reference:exampleTripletXYZ(false,false,true),Options:options,
  Targets:[]tripTargetRequest{
   {Name:"S-Me.xyz",XYZ:exampleTripletXYZ(false,true,true)},
   {Name:"S-Et.xyz",XYZ:exampleTripletXYZ(true,true,true)},
  },Confirmed:true}
 prepared,err:=prepareTriplet(req);if err!=nil{t.Fatal(err)}
 q.Limits=QueueLimits{Concurrency:2,CPU:4,RAMMB:2400}
 for _,item:=range prepared.Items{addEntry(t,q,*item.Request)}
 if err=q.action("","resume");err!=nil{t.Fatal(err)}
 waitQueue(t,q,func(es []*QueueEntry)bool{
  return len(es)==2&&es[0].State=="Completed"&&es[1].State=="Completed"
 })
 q.mu.Lock()
 es:=append([]*QueueEntry(nil),q.Jobs...)
 q.mu.Unlock()
 for _,entry:=range es{
  if entry.Job.Folder==""||len(entry.Job.CompletedStages)!=3{t.Fatalf("stages lost for %s: %+v",entry.Label,entry.Job)}
  for _,file:=range []string{"transferred_initial.xyz","step1_constrained.out","step1_constrained.xyz","step2_triplet.out","step2_triplet.xyz","step2_triplet.gbw","step3_spin.out","step3_spin.cube","triplet_spin_report.json","triplet_spin_report.csv"}{
   if _,err:=os.Stat(filepath.Join(entry.Job.Folder,file));err!=nil{t.Errorf("%s missing %s",entry.Label,file)}
  }
  res,err:=tripReadResult(entry.Job.Folder,entry.Request)
  if err!=nil{t.Fatal(err)}
  if !res.Converged||!res.CubeAvailable||!res.GBWAvailable||len(res.Spin)!=3||res.EnergyEh==nil||res.S2==nil{t.Fatalf("missing verified artifacts/spin: %+v",res)}
  if res.MinimumVerified||res.FrequencyChecked{t.Fatal("frequency OFF must never assert verified minimum")}
  for _,row:=range res.Spin{
   if row.Element!="C"||row.Mulliken==nil||row.Loewdin==nil||row.Hirshfeld==nil{
    t.Errorf("mapped %s spin missing: %+v",row.Role,row)
   }
   expected:=float64(row.Index)*0.009
   if row.Mulliken!=nil&&math.Abs(*row.Mulliken-expected)>1e-8{t.Fatalf("wrong C5/C6 spin identity %s: %+v",row.Role,row)}
  }
  if len(res.SOMOCandidates)==0{t.Fatal("alpha/beta orbital occupations were not parsed")}
 }
 if es[0].Job.Folder==es[1].Job.Folder{t.Fatal("multi-Target jobs shared a directory")}
}

func TestTripletSameTargetEnergyComparisonNeverCrossSpecies(t *testing.T){
 t.Setenv("ORCA_NOVA_TRIPLET_MOCK","1")
 q:=mockQueue(t)
 opts:=standardTripOpts();opts.Cores=2;opts.MaxCore=200
 prepared,err:=prepareTriplet(tripPrepareRequest{Reference:exampleTripletXYZ(false,false,true),Options:opts,
  Targets:[]tripTargetRequest{{Name:"same.xyz",XYZ:exampleTripletXYZ(true,true,true)}},Confirmed:true})
 if err!=nil{t.Fatal(err)}
 q.Limits=QueueLimits{1,2,1800}
 e:=addEntry(t,q,*prepared.Items[0].Request)
 if err:=q.action("","resume");err!=nil{t.Fatal(err)}
 waitQueue(t,q,func(es []*QueueEntry)bool{return es[0].State=="Completed"})
 result,err:=tripReadResult(e.Job.Folder,e.Request);if err!=nil{t.Fatal(err)}
 output:="SCF CONVERGED AFTER 12 CYCLES\nTHE OPTIMIZATION HAS CONVERGED\nFINAL SINGLE POINT ENERGY -75.123000\nExpectation value of <S**2> : 2.030\nORCA TERMINATED NORMALLY\n"
 originalInp:=strings.ReplaceAll(e.Request.Files[2].Content,"Opt","Opt")
 comp,err:=tripCompareResult(exampleTripletXYZ(true,false,true),output,originalInp,nil,result,false)
 if err!=nil{t.Fatal(err)}
 if !comp.SameTarget||comp.DeltaEnergyKcal==nil{t.Fatalf("same Target numerical comparison refused: %+v",comp)}
 if math.Abs(*comp.DeltaEnergyKcal-(float64(-75.1234+75.123)*ehToKcal))>0.0001{t.Fatal("wrong relative energy",comp)}
 _,err=tripCompareResult(exampleTripletXYZ(false,false,true),output,originalInp,nil,result,false)
 if err==nil{t.Fatal("S-Me vs S-Et absolute energy comparison must be forbidden")}
 mixedMethod:=strings.Replace(originalInp,"B3LYP","PBE0",1)
 comp,err=tripCompareResult(exampleTripletXYZ(true,false,true),output,mixedMethod,nil,result,false)
 if err!=nil{t.Fatal(err)}
 if comp.DeltaEnergyKcal!=nil{t.Fatal("mixed Method should not have relative energy")}
}

func TestTripletMissingSpinCubeStopsWorkflowWithFailureAnalysis(t *testing.T){
 t.Setenv("ORCA_NOVA_TRIPLET_MOCK","1")
 t.Setenv("ORCA_NOVA_TRIPLET_NO_CUBE","1")
 q:=mockQueue(t)
 opts:=standardTripOpts();opts.Cores=2;opts.MaxCore=200
 prepared,err:=prepareTriplet(tripPrepareRequest{Reference:exampleTripletXYZ(false,false,true),Options:opts,
  Targets:[]tripTargetRequest{{Name:"no_cube.xyz",XYZ:exampleTripletXYZ(true,false,true)}},Confirmed:true})
 if err!=nil{t.Fatal(err)}
 q.Limits=QueueLimits{1,2,1800}
 entry:=addEntry(t,q,*prepared.Items[0].Request)
 if err=q.action("","resume");err!=nil{t.Fatal(err)}
 waitQueue(t,q,func(es []*QueueEntry)bool{return es[0].State=="Failed"})
 if len(entry.Job.CompletedStages)!=2{t.Fatalf("failed stage marked completed: %v",entry.Job.CompletedStages)}
 report,err:=os.ReadFile(filepath.Join(entry.Job.Folder,"FAILURE_ANALYSIS.txt"));if err!=nil{t.Fatal(err)}
 if !strings.Contains(string(report),"Spin Density CUBE"){t.Fatal("failure analysis missing true reason:",string(report))}
}
func TestTripletSOMOCandidatesAndMethodNormalization(t *testing.T){
 example:=fmt.Sprint("ORBITAL ENERGIES\nSPIN UP ORBITALS\nNO OCC E(Eh) E(eV)\n 0 1.000 -1.000 -27.2\n 1 1.000 -0.300 -8.1\nSPIN DOWN ORBITALS\n 0 1.000 -0.900 -24.4\n 1 0.000 0.100 2.7\n")
 orbital:=tripSOMOCandidates(example)
 if len(orbital)!=1||orbital[0].OrbitalIndex!=1{t.Fatalf("singly occupied candidates: %+v",orbital)}
 a:=tripMethodFingerprint("! UKS B3LYP D4 def2-TZVP TightSCF Opt SMD(acetonitrile)")
 b:=tripMethodFingerprint("! UKS B3LYP D4 def2-TZVP TightSCF SP SMD(acetonitrile)")
 c:=tripMethodFingerprint("! RKS B3LYP D4 def2-TZVP TightSCF Opt Freq SMD(acetonitrile)")
 if a!=b||b!=c{t.Fatal("method normalization incorrectly includes job type")}
}
