package main

import (
 "bytes"
 "os"
 "path/filepath"
 "runtime"
 "strings"
 "testing"
)

// An isolated end-to-end copy/rollback test using REAL binaries from the
// published v1.5.1 Stable and v1.6.0-rc.2 Pre-release ZIP archives.
// The proprietary ORCA executable is NOT part of either package.
// It validates updateScript's file transaction; it is not proof of the
// in-app stable update check or live WebView2 UI interaction.
func TestWindowsPublishedVersionSwapAndRollback(t *testing.T) {
 if runtime.GOOS!="windows" { t.Skip("Windows-only updater integration") }
 oldZip:=os.Getenv("ORCA_PREVIOUS_RELEASE_ZIP")
 nextZip:=os.Getenv("ORCA_CANDIDATE_RELEASE_ZIP")
 if oldZip==""||nextZip=="" { t.Skip("Published release ZIPs not supplied") }
 oldZip,err:=filepath.Abs(oldZip);if err!=nil{t.Fatal(err)}
 nextZip,err=filepath.Abs(nextZip);if err!=nil{t.Fatal(err)}
 testFiles:=func(dir string)map[string][]byte{
  t.Helper()
  files:=map[string][]byte{}
  err:=filepath.Walk(dir,func(path string,info os.FileInfo,err error)error{
   if err!=nil{return err}
   if info.IsDir(){return nil}
   rel,err:=filepath.Rel(dir,path);if err!=nil{return err}
   b,err:=os.ReadFile(path);if err!=nil{return err}
   files[rel]=b
   return nil
  })
  if err!=nil{t.Fatal(err)}
  return files
 }
 stage:=t.TempDir()
 if err:=stageReleaseArchive(nextZip,stage);err!=nil{t.Fatalf("cannot unpack real candidate release: %v",err)}
 candidateFiles:=testFiles(stage)
 if _,ok:=candidateFiles["ORCA_Nova.exe"];!ok{t.Fatal("candidate backend missing")}
 if _,ok:=candidateFiles["ORCA_Nova_UI.exe"];!ok{t.Fatal("candidate UI host missing")}
 system:=os.Getenv("SystemRoot")
 if system==""{system=`C:\Windows`}
 whereExe:=filepath.Join(system,"System32","where.exe")
 if _,err:=os.Stat(whereExe);err!=nil{t.Fatal(err)}
 for _,failAfterInstall:=range []bool{false,true}{
  label:="apply"
  if failAfterInstall{label="rollback"}
  t.Run(label,func(t *testing.T){
   dest:=t.TempDir()
   if err:=stageReleaseArchive(oldZip,dest);err!=nil{t.Fatalf("cannot unpack real v1.5.1 release: %v",err)}
   originalFiles:=testFiles(dest)
   if bytes.Equal(originalFiles["ORCA_Nova.exe"],candidateFiles["ORCA_Nova.exe"]){
    t.Fatal("old and new published backend EXEs unexpectedly identical")
   }
   updateTestWrite(t,dest,"config.json",[]byte("existing personal settings marker"))
   launch:=whereExe
   if failAfterInstall {launch=filepath.Join(dest,"missing-startup-exe.exe")}
   log,err:=updateTestApply(t,stage,dest,launch)
   if failAfterInstall&&err==nil{t.Fatalf("failed launch incorrectly succeeded: %s",log)}
   if !failAfterInstall&&err!=nil{t.Fatalf("real published update failed: %v; %s",err,log)}
   wantFiles:=candidateFiles
   if failAfterInstall{wantFiles=originalFiles}
   for rel,want:=range wantFiles{
    actual,err:=os.ReadFile(filepath.Join(dest,rel))
    if err!=nil||!bytes.Equal(actual,want){
     t.Fatalf("%s: %s does not match expected published release bytes: %v",label,rel,err)
    }
   }
   if failAfterInstall{
    // No freshly introduced candidate file may survive a failed upgrade.
    for rel:=range candidateFiles {
     if _,wasPresent:=originalFiles[rel];wasPresent{continue}
     if _,err:=os.Stat(filepath.Join(dest,rel));!os.IsNotExist(err){
      t.Fatalf("new candidate file survived rollback: %s (%v)",rel,err)
     }
    }
    if !strings.Contains(log,"rollback: SUCCESS"){t.Fatalf("rollback not reported: %s",log)}
   }else if !strings.Contains(log,"SUCCESS:"){t.Fatalf("upgrade not reported: %s",log)}
   if string(updateTestRead(t,dest,"config.json"))!="existing personal settings marker"{
    t.Fatal("personal settings were changed")
   }
   backups,err:=filepath.Glob(filepath.Join(dest,".ORCA_Nova_backup_*"))
   if err!=nil||len(backups)!=1{t.Fatalf("expected one retained backup folder, got %v (%v)",backups,err)}
   for rel,old:=range originalFiles{
    // Files included in the candidate's copy list must have a matching
    // original backup when they existed before upgrading.
    if _,candidate:=candidateFiles[rel];!candidate{continue}
    backed,err:=os.ReadFile(filepath.Join(backups[0],rel))
    if err!=nil||!bytes.Equal(backed,old){t.Fatalf("original backup damaged: %s (%v)",rel,err)}
   }
  })
 }
}
