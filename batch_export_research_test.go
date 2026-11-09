package main

import (
 "bytes"
 "encoding/csv"
 "net/http"
 "runtime"
 "time"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestBatchScientificExportsRespectSourceAndConvergence(t *testing.T) {
 q:=mockQueue(t)
 req:=smallReq("=UNTRUSTED()")
 e:=addEntry(t,q,req)
 q.mu.Lock()
 err:=q.prepareLocked(e,q.app.cfg)
 q.mu.Unlock()
 if err!=nil {t.Fatal(err)}
 stage:=req.Stages[0]
 path:=filepath.Join(e.Job.Folder,strings.TrimSuffix(stage.Input,filepath.Ext(stage.Input))+".out")
 if err=os.WriteFile(path,[]byte(sampleORCA),0600);err!=nil{t.Fatal(err)}
 e.State="Completed"
 var raw bytes.Buffer
 q.exportLocked(&raw)
 records,err:=csv.NewReader(&raw).ReadAll()
 if err!=nil {t.Fatal(err)}
 if len(records)!=len(req.Stages)+1 {t.Fatalf("unexpected stage count: %d",len(records))}
 head:=map[string]int{}
 for i,name:=range records[0] {head[name]=i}
 get:=func(row []string,name string)string {t.Helper();i,ok:=head[name];if !ok {t.Fatalf("missing column %s",name)};return row[i]}
 valid,missing:=records[1],records[2]
 if !strings.HasPrefix(get(valid,"Molecule"),"'=") {t.Fatal("CSV spreadsheet formula injection unsafe",valid)}
 if get(valid,"ORCA_Normal")!="true"||get(valid,"SCF_Converged")!="true" {
  t.Fatalf("unconverged sample exported as valid: %v",valid)
 }
 if get(valid,"Gibbs_Eh")!="-151.5461827" {t.Fatal(get(valid,"Gibbs_Eh"))}
 if get(valid,"ZPE_Correction_Eh")!="0.02631509" {t.Fatal(get(valid,"ZPE_Correction_Eh"))}
 if get(valid,"Vertical_S1_eV")!="2.3390"||get(valid,"Vertical_T1_eV")!="1.9410" {
  t.Fatal("wrong vertical roots",get(valid,"Vertical_S1_eV"),get(valid,"Vertical_T1_eV"))
 }
 if get(valid,"Vertical_S1_kcal_mol")=="Not calculated" {t.Fatal("missing kcal/mol conversion")}
 if get(missing,"Electronic_Eh")!="Not calculated"||get(missing,"Vertical_T1_eV")!="Not calculated" {
  t.Fatal("missing OUT leaked a previous stage's values",missing)
 }
 var text bytes.Buffer
 q.exportHumanTextLocked(&text)
 for _,key:=range []string{"Batch Research Summary","Vertical S1","kcal/mol","Gibbs:","Not calculated","ORCA 0"} {
  if !strings.Contains(text.String(),key) {t.Errorf("TXT missing %q",key)}
 }
 if strings.Contains(text.String(),"Adiabatic T1−S0: +") {
  t.Fatal("TXT invented unverified adiabatic state difference")
 }

 // Once a stage is incomplete, even a residual numerical value must not
 // masquerade as a validated scientific result.
 failOut:="FINAL SINGLE POINT ENERGY -75.0\nTD-DFT EXCITED STATES (SINGLETS)\nSTATE  1: E= 0.100000 au 2.721 eV 21000 cm**-1\n"
 if err=os.WriteFile(path,[]byte(failOut),0600);err!=nil {t.Fatal(err)}
 raw.Reset();q.exportLocked(&raw)
 records,err=csv.NewReader(&raw).ReadAll();if err!=nil{t.Fatal(err)}
 if got:=get(records[1],"Electronic_Eh");got!="-151.55083691" && got!="Failed" {
  t.Fatalf("unexpected failed status=%q",got)
 }
 if get(records[1],"Vertical_S1_eV")!="Not calculated" {
  t.Fatal("incomplete OUT leaked numerical excitation")
 }
}

func TestBatchExportTXTEndpointAndInvalidFormat(t *testing.T) {
 q:=mockQueue(t)
 a:=q.app
 a.token="batch-token"
 for _,mode:=range []struct{format string;status int;fragment string}{
  {"txt",200,"Batch Research Summary"},
  {"csv",200,"Molecule,Status,Stage"},
  {"unknown",400,"지원하는 보고서 형식"},
 }{
  r:=httptest.NewRequest(http.MethodGet,"/api/queue/export?format="+mode.format+"&session=batch-token",nil)
  w:=httptest.NewRecorder();a.guard(a.route)(w,r)
  if w.Code!=mode.status||!strings.Contains(w.Body.String(),mode.fragment) {
   t.Fatalf("format=%q status=%d body=%s",mode.format,w.Code,w.Body.String())
  }
  if mode.status==200&&w.Header().Get("Cache-Control")!="no-store" {
   t.Fatal("research export must be private and noncacheable")
  }
 }
 w:=httptest.NewRecorder();a.guard(a.route)(w,httptest.NewRequest(http.MethodGet,"/api/queue/export?format=txt",nil))
 if w.Code!=http.StatusForbidden {t.Fatal("unauthenticated TXT export",w.Code)}
}

// Exercise four truly concurrent mock process slots on machines with at least
// four logical CPUs, and verify one cancellation does not kill the other three.
// Actual research-PC ORCA/MPI validation is a separate acceptance gate.
func TestFourParallelIndependentCancellation(t *testing.T) {
 if runtime.NumCPU()<4 {t.Skip("requires four logical CPUs to test four concurrent workers")}
 q:=mockQueue(t)
 q.Limits=QueueLimits{Concurrency:4,CPU:4,RAMMB:500}
 for _,name:=range []string{"A","B","C","D"} {
  r:=smallReq(name)
  r.Files[1].Content+="\n# MOCK_BLOCK\n"
  addEntry(t,q,r)
 }
 if err:=q.action("","resume");err!=nil{t.Fatal(err)}
 waitQueue(t,q,func(es []*QueueEntry)bool{
  for _,e:=range es{if e.State!="Running" {return false}}
  return true
 })
 q.mu.Lock()
 var cpu,ram int
 dirs:=map[string]bool{}
 for _,e:=range q.Jobs {
  cpu+=e.CPU;ram+=e.RAMMB
  if dirs[e.Job.Folder] {q.mu.Unlock();t.Fatal("Job output folder overlap")}
  dirs[e.Job.Folder]=true
 }
 cancelID:=q.Jobs[0].ID
 q.mu.Unlock()
 if cpu>q.Limits.CPU||ram>q.Limits.RAMMB {t.Fatal("Scheduler overcommitted CPU/RAM")}
 // Explicitly wait for all four mock children to start, not just Queue state.
 q.mu.Lock()
 folders:=make([]string,0,4)
 for _,e:=range q.Jobs {folders=append(folders,e.Job.Folder)}
 q.mu.Unlock()
 for _,folder:=range folders {
  deadline:=time.Now().Add(5*time.Second)
  for {
   if _,err:=os.Stat(filepath.Join(folder,"started"));err==nil{break}
   if time.Now().After(deadline){t.Fatal("mock subprocess never started: "+folder)}
   time.Sleep(15*time.Millisecond)
  }
 }
 if err:=q.action(cancelID,"cancel");err!=nil{t.Fatal(err)}
 waitQueue(t,q,func(es []*QueueEntry)bool{
  if es[0].State!="Interrupted"{return false}
  for _,e:=range es[1:]{if e.State!="Running"{return false}}
  return true
 })
 if err:=os.WriteFile(filepath.Join(q.app.cfg.WorkRoot,"allow_continue"),[]byte("yes"),0600);err!=nil{t.Fatal(err)}
 waitQueue(t,q,func(es []*QueueEntry)bool{
  if es[0].State!="Interrupted" {return false}
  for _,e:=range es[1:]{if e.State!="Completed"{return false}}
  return true
 })
}
