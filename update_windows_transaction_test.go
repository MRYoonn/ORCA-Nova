package main

import (
 "bytes"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "strings"
 "testing"
)
func updateTestWrite(t *testing.T,dir,name string,data []byte){
 t.Helper()
 p:=filepath.Join(dir,name)
 if err:=os.MkdirAll(filepath.Dir(p),0700);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(p,data,0600);err!=nil{t.Fatal(err)}
}
func updateTestRead(t *testing.T,dir,name string)[]byte{
 t.Helper()
 d,err:=os.ReadFile(filepath.Join(dir,name));if err!=nil{t.Fatal(err)}
 return d
}
func updateTestApply(t *testing.T,stage,target,launch string)(string,error) {
 t.Helper()
 ps:=filepath.Join(t.TempDir(),"apply.ps1")
 log:=filepath.Join(t.TempDir(),"update.log")
 if err:=os.WriteFile(ps,[]byte(updateScript),0600);err!=nil{t.Fatal(err)}
 cmd:=exec.Command("powershell.exe","-NoProfile","-ExecutionPolicy","Bypass","-File",ps,
  "-CurrentPid","2147483646","-StageDir",stage,"-TargetDir",target,"-Launch",launch,"-Log",log)
 out,err:=cmd.CombinedOutput()
 text,_:=os.ReadFile(log)
 return string(text)+"\n"+string(out),err
}
func TestWindowsUpdateTransactionalSuccessAndFailureRollback(t *testing.T){
 if runtime.GOOS!="windows" {t.Skip("native Windows PowerShell required")}
 system:=os.Getenv("SystemRoot")
 if system=="" {system=`C:\Windows`}
 whereExe:=filepath.Join(system,"System32","where.exe")
 findExe:=filepath.Join(system,"System32","find.exe")
 updatedExe,err:=os.ReadFile(whereExe);if err!=nil{t.Fatal(err)}
 originalExe,err:=os.ReadFile(findExe);if err!=nil{t.Fatal(err)}
 for _,scenario:=range []struct{name string;launchGood bool}{{"success",true},{"missing-launch-restores-all",false}} {
 t.Run(scenario.name,func(t *testing.T){
  target:=t.TempDir();stage:=t.TempDir()
  updateTestWrite(t,target,"ORCA_Nova.exe",originalExe)
  updateTestWrite(t,target,"ORCA_Nova_UI.exe",[]byte("old UI"))
  updateTestWrite(t,target,"runtime/WebView2Loader.dll",[]byte("old DLL"))
  updateTestWrite(t,target,"settings.json",[]byte("user preferences"))
  updateTestWrite(t,stage,"ORCA_Nova.exe",updatedExe)
  updateTestWrite(t,stage,"ORCA_Nova_UI.exe",[]byte("new UI"))
  updateTestWrite(t,stage,"runtime/WebView2Loader.dll",[]byte("new DLL"))
  updateTestWrite(t,stage,"runtime/NewDependency.dll",[]byte("added DLL"))
  launch:=filepath.Join(target,"ORCA_Nova.exe")
  if !scenario.launchGood {launch=filepath.Join(target,"missing-startup.exe")}
  log,err:=updateTestApply(t,stage,target,launch)
  if scenario.launchGood {
    if err!=nil{t.Fatalf("full update failed: %v; %s",err,log)}
    if !bytes.Equal(updateTestRead(t,target,"ORCA_Nova.exe"),updatedExe){t.Fatal("backend EXE not replaced")}
    if string(updateTestRead(t,target,"ORCA_Nova_UI.exe"))!="new UI"{t.Fatal("UI EXE not replaced")}
    if string(updateTestRead(t,target,"runtime/WebView2Loader.dll"))!="new DLL"{t.Fatal("UI DLL not replaced")}
    if string(updateTestRead(t,target,"runtime/NewDependency.dll"))!="added DLL"{t.Fatal("new file not added")}
    if !strings.Contains(log,"SUCCESS:"){t.Fatal("missing success log",log)}
  }else{
    if err==nil{t.Fatal("missing launch was reported as successful:",log)}
    if !bytes.Equal(updateTestRead(t,target,"ORCA_Nova.exe"),originalExe){t.Fatal("backend rollback failed")}
    if string(updateTestRead(t,target,"ORCA_Nova_UI.exe"))!="old UI"{t.Fatal("native UI rollback failed")}
    if string(updateTestRead(t,target,"runtime/WebView2Loader.dll"))!="old DLL"{t.Fatal("nested DLL rollback failed")}
    if _,err:=os.Stat(filepath.Join(target,"runtime","NewDependency.dll"));!os.IsNotExist(err){
       t.Fatalf("new dependency not removed on rollback: %v",err)
    }
    if !strings.Contains(log,"rollback: SUCCESS"){t.Fatal("full rollback not acknowledged",log)}
  }
  if string(updateTestRead(t,target,"settings.json"))!="user preferences"{
    t.Fatal("unrelated user settings were modified")
  }
  backs,err:=filepath.Glob(filepath.Join(target,".ORCA_Nova_backup_*"));if err!=nil||len(backs)!=1{
    t.Fatal(fmt.Sprintf("expected exactly one retained backup folder: %v %v",backs,err))
  }
  if !bytes.Equal(updateTestRead(t,backs[0],"ORCA_Nova.exe"),originalExe){t.Fatal("retained backup invalid")}
 })
 }
}
