package main

import (
 "math"
 "strings"
 "testing"
)

func profileAnalysis(name, method string, mult int, energy float64) OrcaAnalysis {
 n:=0
 temp:=298.15
 return OrcaAnalysis{
  Filename:name, Method:method, Geometry:name+".xyz",Charge:0,Multiplicity:mult,
  Normal:true,SCFConverged:true,Optimized:true,Electronic:&energy,Gibbs:&energy,
  ImaginaryCount:&n,Temperature:&temp,
 }
}
func TestScientificAdiabaticValidations(t *testing.T) {
 s:=profileAnalysis("s0.out","PBE0 def2-TZVP D4 TightSCF Opt",1,-100.0)
 tr:=profileAnalysis("t1.out","PBE0 def2-TZVP D4 TightSCF Opt Freq UKS",3,-99.9)
 refs:=map[string]OrcaAnalysis{"s0.out":s,"t1.out":tr}
 req:=ScientificProfileRequest{Kind:"adiabatic",Energy:"Electronic",Confirmed:true,Sources:map[string]string{"S0":"s0.out","T1":"t1.out"}}
 res,err:=scientificProfile(req,refs);if err!=nil{t.Fatal(err)}
 if res.GapKJ==nil||math.Abs(*res.GapKJ-262.549963948)>1e-4 {t.Fatalf("gap: %+v",res)}
 if len(res.Points)!=2||res.Points[0].Y!=0{t.Fatalf("profile: %+v",res)}
 req.Confirmed=false;if _,err=scientificProfile(req,refs);err==nil{t.Fatal("must require identity confirmation")}
 req.Confirmed=true;tr.Method="B3LYP def2-TZVP D4 Opt";refs["t1.out"]=tr
 if _,err=scientificProfile(req,refs);err==nil{t.Fatal("different method accepted")}
 tr.Method=s.Method;tr.Multiplicity=1;refs["t1.out"]=tr
 if _,err=scientificProfile(req,refs);err==nil{t.Fatal("wrong state accepted")}
}

func TestReactionProfileFrequencyAndSettings(t *testing.T) {
 r:=profileAnalysis("r.out","PBE0 def2-SVP D4 Opt Freq",1,-200.0)
 ts:=profileAnalysis("ts.out","PBE0 def2-SVP D4 OptTS Freq",1,-199.95)
 p:=profileAnalysis("p.out","PBE0 def2-SVP D4 Opt Freq",1,-200.06)
 one:=1;ts.ImaginaryCount=&one
 inputs:=map[string]OrcaAnalysis{"r.out":r,"ts.out":ts,"p.out":p}
 req:=ScientificProfileRequest{Kind:"reaction",Energy:"Gibbs",Confirmed:true,Sources:map[string]string{"Reactant":"r.out","TS":"ts.out","Product":"p.out"}}
 got,err:=scientificProfile(req,inputs);if err!=nil{t.Fatal(err)}
 if got.BarrierKJ==nil||*got.BarrierKJ<131||*got.BarrierKJ>132 {t.Fatal(got)}
 if got.ReactionKJ==nil||*got.ReactionKJ> -157 ||*got.ReactionKJ< -158 {t.Fatal(got)}
 ts.ImaginaryCount=nil;inputs["ts.out"]=ts
 if _,err:=scientificProfile(req,inputs);err==nil{t.Fatal("TS without frequency passed")}
 ts.ImaginaryCount=&one;ts.Gibbs=nil;inputs["ts.out"]=ts
 if _,err:=scientificProfile(req,inputs);err==nil{t.Fatal("missing Gibbs passed")}
}
func TestScanSpinOnlyRealStepRows(t *testing.T) {
 out:=strings.Join([]string{
  "RELAXED SURFACE SCAN STEP 1","SCAN COORDINATE: 1.30","SCF CONVERGED",
  "FINAL SINGLE POINT ENERGY -100.00","THE OPTIMIZATION HAS CONVERGED",
  "MULLIKEN ATOMIC CHARGES AND SPIN POPULATIONS",
  " 4 C : -0.10 0.51"," 5 C : -0.12 -0.31",
  "RELAXED SURFACE SCAN STEP 2","SCAN COORDINATE: 1.40","SCF CONVERGED",
  "FINAL SINGLE POINT ENERGY -100.01","THE OPTIMIZATION HAS CONVERGED",
 }, "\n")
 v:=parseVisualization("scan.out",out,"! PBE0 def2-SVP Opt\n%geom\n Scan\n B 4 5 = 1.3,1.4,2\n end\nend\n")
 if len(v.PES)!=2 {t.Fatalf("steps %d",len(v.PES))}
 if v.PES[0].Analysis==nil||len(v.PES[0].Analysis.Populations["Mulliken"])!=2 {t.Fatal("missing step 1 actual spin")}
 if v.PES[1].Analysis!=nil {t.Fatal("invented step 2 spin")}
 if v.PES[1].Coordinate==nil||*v.PES[1].Coordinate!=1.4{t.Fatal("coordinate mismatch")}
}
