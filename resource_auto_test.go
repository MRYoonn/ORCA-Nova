package main

import (
 "strings"
 "testing"
)

func resourceTestRequest(label string) RunRequest {
 return RunRequest{Label:label,Files:[]JobFile{
  {Name:"mol.xyz",Content:"2\nH2\nH 0 0 0\nH 0 0 0.74\n"},
  {Name:"first.inp",Content:"! PBE0 def2-SVP Opt\n\n%pal\n  nprocs 8\nend\n\n%maxcore 2500\n* xyzfile 0 1 mol.xyz\n"},
  {Name:"second.inp",Content:"! PBE0 def2-SVP SP\r\n%pal\r\n  nprocs 8\r\nend\r\n%maxcore 2500\r\n* xyzfile 0 1 first.xyz\r\n"},
 },Stages:[]Stage{{Name:"Opt",Input:"first.inp",Source:"mol.xyz",Opt:true},{Name:"SP",Input:"second.inp",Source:"first.xyz"}}}
}

func TestAutoResourceAllocationFourJobs(t *testing.T) {
 sys:=resourceSystem{CPU:16,TotalMB:65536,AvailableMB:62500,BudgetMB:50000,Supported:true}
 jobs:=[]RunRequest{resourceTestRequest("a"),resourceTestRequest("b"),resourceTestRequest("c"),resourceTestRequest("d")}
 out,limits,err:=planAutoResources(jobs,[]bool{true,true,true,true},16,sys)
 if err!=nil {t.Fatal(err)}
 if limits!=(QueueLimits{Concurrency:4,CPU:16,RAMMB:50000}) {t.Fatal(limits)}
 totalRAM:=0
 for i,req:=range out{
  c,r,err:=resources(req)
  if err!=nil||c!=4||r!=12500{t.Fatalf("job %d: CPU=%d RAM=%d %v",i,c,r,err)}
  totalRAM+=r
  if !strings.Contains(req.Files[1].Content,"! PBE0 def2-SVP Opt\n")||!strings.Contains(req.Files[2].Content,"  nprocs 4\r\n")||req.Files[0].Content!=jobs[i].Files[0].Content{t.Fatal("non-resource contents or line endings changed")}
  if !strings.Contains(jobs[i].Files[1].Content,"nprocs 8"){t.Fatal("source request mutated")}
 }
 if totalRAM>limits.RAMMB{t.Fatal("overcommitted")}
}
func TestAutoResourceUnevenCPUAndFixedInputs(t *testing.T){
 sys:=resourceSystem{CPU:16,TotalMB:65536,AvailableMB:62500,BudgetMB:50000,Supported:true}
 jobs:=[]RunRequest{resourceTestRequest("a"),resourceTestRequest("b"),resourceTestRequest("c")}
 out,limits,err:=planAutoResources(jobs,[]bool{true,true,true},10,sys)
 if err!=nil||limits.Concurrency!=3{t.Fatal(err,limits)}
 sum:=0;ram:=0
 for i,r:=range out{c,m,e:=resources(r);if e!=nil||c!=[]int{4,3,3}[i]{t.Fatal(c,m,e)};sum+=c;ram+=m}
 if sum!=10||ram>50000{t.Fatal(sum,ram)}
 out,_,err=planAutoResources(jobs,[]bool{true,false,true},10,sys)
 if err!=nil||out[1].Files[1].Content!=jobs[1].Files[1].Content{t.Fatal("fixed input changed",err)}
}
func TestAutoResourceLowMemoryFailsClosed(t *testing.T){
 jobs:=[]RunRequest{resourceTestRequest("a")}
 for _,sys:=range []resourceSystem{{CPU:16},{CPU:16,Supported:true,BudgetMB:639}} {
  if _,_,err:=planAutoResources(jobs,[]bool{true},16,sys);err==nil{t.Fatal("invalid memory accepted")}
 }
 if memoryBudget(65536,62500)!=50000||memoryBudget(16000,1100)!=76||memoryBudget(10,20)!=0{t.Fatal("RAM reserve policy incorrect")}
 sys:=resourceSystem{CPU:16,BudgetMB:2560,Supported:true}
 out,_,err:=planAutoResources(jobs,[]bool{true},16,sys)
 if err!=nil{t.Fatal(err)}
 c,r,e:=resources(out[0]);if e!=nil||c!=4||r!=2560{t.Fatal(c,r,e)}
}
func TestAutoResourceRejectsNoncanonicalPAL(t *testing.T){
 req:=resourceTestRequest("a")
 req.Files[1].Content=strings.Replace(req.Files[1].Content,"%pal\n  nprocs 8\nend","%pal\n  nprocs 8\n  nprocs_group 2\nend",1)
 if _,err:=autoRewrite(req,2,1000);err==nil{t.Fatal("noncanonical input rewritten")}
}

func TestPendingResourceReallocationUsesRemainingBudget(t *testing.T){
 req:=resourceTestRequest("next")
 sys:=resourceSystem{CPU:16,Supported:true,BudgetMB:50000}
 first,c,r,err:=replanPendingAllocation(req,16,50000,4,4,sys)
 if err!=nil||c!=4||r!=12500{t.Fatal(c,r,err)}
 next,c2,r2,err:=replanPendingAllocation(req,12,37500,3,3,sys)
 if err!=nil||c2!=4||r2!=12500{t.Fatal(c2,r2,err)}
 last,c3,r3,err:=replanPendingAllocation(req,16,50000,4,1,sys)
 if err!=nil||c3!=16||r3!=50000{t.Fatal(c3,r3,err)}
 if !strings.Contains(first.Files[1].Content,"nprocs 4")||!strings.Contains(next.Files[1].Content,"nprocs 4")||!strings.Contains(last.Files[1].Content,"nprocs 16"){t.Fatal("resource directive incorrect")}
 if !strings.Contains(req.Files[1].Content,"nprocs 8"){t.Fatal("source modified")}
}
func TestPendingResourceReallocationRespectsLiveRAM(t *testing.T){
 req:=resourceTestRequest("next")
 sys:=resourceSystem{CPU:16,Supported:true,BudgetMB:1280}
 next,c,r,err:=replanPendingAllocation(req,16,50000,4,2,sys)
 if err!=nil||c!=1||r!=640{t.Fatal(c,r,err)}
 if !strings.Contains(next.Files[1].Content,"nprocs 1"){t.Fatal("incorrect reduced allocation")}
 sys.BudgetMB=639
 if _,_,_,err:=replanPendingAllocation(req,16,50000,4,2,sys);err==nil{t.Fatal("unsafe launch with insufficient physical RAM")}
}

func TestAutoResourceTripletPreservesScientificWorkflow(t *testing.T) {
 prepared,err:=prepareTriplet(tripPrepareRequest{
  Reference:exampleTripletXYZ(false,false,true),
  Targets:[]tripTargetRequest{{Name:"S-Me.xyz",XYZ:exampleTripletXYZ(false,false,true)},{Name:"S-Et.xyz",XYZ:exampleTripletXYZ(true,true,true)}},
  Options:standardTripOpts(),Confirmed:true,
 })
 if err!=nil{t.Fatal(err)}
 jobs:=[]RunRequest{*prepared.Items[0].Request,*prepared.Items[1].Request}
 sys:=resourceSystem{CPU:16,Supported:true,BudgetMB:50000}
 planned,limits,err:=planAutoResources(jobs,[]bool{true,true},16,sys)
 if err!=nil||limits.Concurrency!=2{t.Fatal(limits,err)}
 for i,p:=range planned {
  if !p.Triplet.Confirmed || len(p.Stages)!=3 || p.Files[0].Content!=jobs[i].Files[0].Content{t.Fatal("Triplet mapping or geometry changed")}
  if !preflight(p).OK {t.Fatal(preflight(p).Errors)}
  c,r,e:=resources(p);if e!=nil||c!=8||r!=25000{t.Fatal(c,r,e)}
  strip:=func(s string)string{return autoMaxCoreLine.ReplaceAllString(autoPALBlock.ReplaceAllString(s,"PAL"),"MAXCORE")}
  for k:=1;k<len(p.Files);k++{
   if strip(p.Files[k].Content)!=strip(jobs[i].Files[k].Content){t.Fatal("Non-resource INP changed",p.Files[k].Name)}
  }
  if !strings.Contains(p.Files[3].Content,`%moinp "step2_triplet.gbw"`)||!strings.Contains(p.Files[1].Content,"Constraints"){t.Fatal("Triplet scientific workflow changed")}
 }
 next,c,r,e:=replanPendingAllocation(planned[1],16,50000,4,1,sys)
 if e!=nil||c!=16||r!=50000|| !preflight(next).OK{t.Fatal(c,r,e)}
 if !strings.Contains(next.Files[3].Content,"MOREAD"){t.Fatal("Internal GBW reuse lost")}
}

func TestCPUUsageCounterDelta(t *testing.T){
 before:=cpuCounters{idle:100,kernel:400,user:300}
 // kernel delta = 200 including idle delta = 120; user delta = 100
 after:=cpuCounters{idle:220,kernel:600,user:400}
 pct,ok:=cpuUsageDelta(before,after);if !ok||pct!=60{t.Fatal(pct,ok)}
 for _,bad:=range []cpuCounters{before,{idle:99,kernel:600,user:400},{idle:999,kernel:600,user:400}}{
  if _,ok:=cpuUsageDelta(before,bad);ok{t.Fatal("invalid CPU counter accepted",bad)}
 }
 if pct,ok:=cpuUsageDelta(before,cpuCounters{idle:400,kernel:700,user:300});!ok||pct!=0{t.Fatal(pct,ok)}
}
