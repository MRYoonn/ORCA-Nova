package main

import (
 "strconv"
 "strings"
 "testing"
)
const waterXYZ="3\nwater from Avogadro\nO 0 0 0\nH 0.757 0.586 0\nH -0.757 0.586 0\n"
func testReq(charge,mult int) RunRequest {
 inp:="! PBE0 def2-SVP Opt\n%pal\n nprocs 16\nend\n%maxcore 3000\n* xyzfile "+strconv.Itoa(charge)+" "+strconv.Itoa(mult)+" water.xyz\n"
 return RunRequest{Files:[]JobFile{{Name:"water.xyz",Content:waterXYZ},{Name:"water_opt.inp",Content:inp}},Stages:[]Stage{{Name:"S0 Opt",Input:"water_opt.inp",Source:"water.xyz",Opt:true}}}
}
func TestPreflightPass(t *testing.T){
 r:=preflight(testReq(0,1));if !r.OK||r.Stages[0].Atoms!=3||r.Stages[0].Electrons!=10{t.Fatalf("valid XYZ rejected: %+v",r)}
}
func TestPreflightElectronParity(t *testing.T){
 if !preflight(testReq(1,2)).OK{t.Fatal("odd electron doublet should pass")}
 if preflight(testReq(1,1)).OK{t.Fatal("odd electron singlet must fail")}
 if preflight(testReq(0,2)).OK{t.Fatal("even electron doublet must fail")}
 if !preflight(testReq(0,3)).OK{t.Fatal("even electron triplet should pass")}
}
func TestPreflightXYZ(t *testing.T){
 for _,bad:=range []string{"3\nbad\nO 0 0 0\nH 0 0 0\n","1\nbad\nXy 0 0 0\n","1\nbad\nC NaN 0 0\n"}{
  r:=testReq(0,1);r.Files[0].Content=bad;if preflight(r).OK{t.Fatalf("accepted bad XYZ: %q",bad)}
 }
}
func TestPreflightMismatch(t *testing.T){
 r:=testReq(0,1);r.Files[1].Content=strings.Replace(r.Files[1].Content,"water.xyz","other.xyz",1);if preflight(r).OK{t.Fatal("XYZ ref mismatch accepted")}
}
func TestPreflightChained(t *testing.T){
 r:=testReq(0,1)
 r.Files=append(r.Files,JobFile{Name:"t1.inp",Content:"! PBE0 def2-SVP Opt\n* xyzfile 0 3 water_opt.xyz\n"})
 r.Stages=append(r.Stages,Stage{Name:"T1 Opt",Input:"t1.inp",Source:"water_opt.xyz",Opt:true})
 if got:=preflight(r);!got.OK{t.Fatalf("valid chained workflow rejected: %+v",got)}
 r.Stages[0].Opt=false;if preflight(r).OK{t.Fatal("missing prerequisite optimization accepted")}
}
func TestPreflightMemoryWarning(t *testing.T){
 r:=testReq(0,1);r.Files[1].Content=strings.Replace(r.Files[1].Content,"%maxcore 3000","%maxcore 5000",1)
 got:=preflight(r);if !got.OK||len(got.Warnings)==0{t.Fatalf("missing warning: %+v",got)}
}
