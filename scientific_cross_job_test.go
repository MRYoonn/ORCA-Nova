package main

import (
 "fmt"
 "os"
 "path/filepath"
 "strings"
 "testing"
)
func writeProfileFixture(t *testing.T,folder string,mult int,energy float64){
 t.Helper()
 inp:=fmt.Sprintf("! PBE0 def2-SVP D4 TightSCF Opt Freq\n* xyzfile 0 %d molecule.xyz\n",mult)
 out:=fmt.Sprintf("SCF CONVERGED\nTHE OPTIMIZATION HAS CONVERGED\nFINAL SINGLE POINT ENERGY %.9f\nORCA TERMINATED NORMALLY\n",energy)
 for name,data:=range map[string]string{"same.inp":inp,"same.out":out}{
  if err:=os.WriteFile(filepath.Join(folder,name),[]byte(data),0600);err!=nil{t.Fatal(err)}
 }
}
func TestResolveProfileSourcesAcrossJobsWithSameFilename(t *testing.T){
 aDir,bDir:=t.TempDir(),t.TempDir()
 writeProfileFixture(t,aDir,1,-101.0)
 writeProfileFixture(t,bDir,3,-100.9)
 app:=&App{queue:&JobQueue{Jobs:[]*QueueEntry{
  {ID:"job-a",Job:&Job{Folder:aDir}},
  {ID:"job-b",Job:&Job{Folder:bDir}},
 }}}
 req:=ScientificProfileRequest{
  Kind:"adiabatic",Energy:"Electronic",Confirmed:true,
  Sources:map[string]string{"S0":"same.out","T1":"same.out"},
  Jobs:map[string]string{"S0":"job-a","T1":"job-b"},
 }
 resolved,sources,err:=app.resolveProfileSources(req)
 if err!=nil{t.Fatalf("source resolution failed: %v",err)}
 if len(sources)!=2||resolved.Sources["S0"]==resolved.Sources["T1"] {
  t.Fatalf("different jobs did not remain distinct: %+v",resolved.Sources)
 }
 got,err:=scientificProfile(resolved,sources)
 if err!=nil{t.Fatalf("valid matching methods failed: %v",err)}
 if got.GapKJ==nil||*got.GapKJ<262||*got.GapKJ>263 {t.Fatalf("incorrect gap: %+v",got)}
 // An unknown job cannot be interpreted as a filesystem path.
 req.Jobs["T1"]=bDir
 if _,_,err:=app.resolveProfileSources(req);err==nil{t.Fatal("arbitrary folder accepted as queue ID")}
 // Explicitly reject one calculation file assigned to both roles.
 req.Jobs["T1"]="job-a"
 if _,_,err:=app.resolveProfileSources(req);err==nil{t.Fatal("same job and OUT reused for both roles")}
 req.Jobs["T1"]="job-b"
 req.Sources["T1"]="../same.out"
 if _,_,err:=app.resolveProfileSources(req);err==nil{t.Fatal("path traversal accepted")}
 req.Sources["T1"]="same.out"
 req.Jobs["invalid-role"]="job-a"
 if _,_,err:=app.resolveProfileSources(req);err==nil{t.Fatal("unexpected role accepted")}
}

func TestCrossJobProfileBlocksMixedMethod(t *testing.T){
 aDir,bDir:=t.TempDir(),t.TempDir()
 writeProfileFixture(t,aDir,1,-101.0)
 writeProfileFixture(t,bDir,3,-100.9)
 if err:=os.WriteFile(filepath.Join(bDir,"same.inp"),[]byte(strings.Replace(
  "! PBE0 def2-SVP D4 TightSCF Opt\n* xyzfile 0 3 molecule.xyz\n","PBE0","B3LYP",1)),0600);err!=nil{t.Fatal(err)}
 app:=&App{queue:&JobQueue{Jobs:[]*QueueEntry{{ID:"a",Job:&Job{Folder:aDir}},{ID:"b",Job:&Job{Folder:bDir}}}}}
 req:=ScientificProfileRequest{Kind:"adiabatic",Energy:"Electronic",Confirmed:true,
 Sources:map[string]string{"S0":"same.out","T1":"same.out"},Jobs:map[string]string{"S0":"a","T1":"b"}}
 resolved,sources,err:=app.resolveProfileSources(req)
 if err!=nil{t.Fatal(err)}
 if _,err:=scientificProfile(resolved,sources);err==nil{t.Fatal("mismatched functional was silently compared")}
}
