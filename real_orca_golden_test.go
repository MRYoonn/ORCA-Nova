package main

import (
 "math"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

// Real ORCA 6.1.1 Golden Test. The original .out/.inp files are NOT checked
// into ORCA Nova: CI downloads an immutable public teaching example and
// deletes the temporary copy after testing. This test is skipped when no
// external source directory has been supplied (regular offline builds).
//
// Upstream repository: dhwan0112/dhk.github.io
// Source commit: 0d7afec3fb8ca34b287e2413fccb4a7376899a75
// Provenance: guides/orca/ko/examples.html, actual ORCA 6.1.1 runs.
func TestRealOrca611WaterGolden(t *testing.T) {
 folder:=os.Getenv("ORCA_REAL_GOLDEN_DIR")
 if folder=="" {t.Skip("ORCA_REAL_GOLDEN_DIR unset; no real ORCA outputs installed")}
 tests:=[]struct{
  name string; dir string; expectedE float64; optimized bool
 }{
  {"SinglePoint","01-single-point",-76.321274411145,false},
  {"OptimizationFrequency","02-opt-freq",-76.426369457144,true},
 }
 for _,tt:=range tests {t.Run(tt.name,func(t *testing.T){
  outPath:=filepath.Join(folder,tt.dir,"water.out")
  inpPath:=filepath.Join(folder,tt.dir,"water.inp")
  out,err:=os.ReadFile(outPath);if err!=nil{t.Fatal(err)}
  inp,err:=os.ReadFile(inpPath);if err!=nil{t.Fatal(err)}
  if !strings.Contains(string(out),"Program Version 6.1.1") {t.Fatal("Golden fixture is not a real ORCA 6.1.1 output")}
  a,err:=ParseOrcaOutput(strings.NewReader(string(out)),"water.out",string(inp))
  if err!=nil{t.Fatal(err)}
  if !a.Normal||!a.SCFConverged {t.Fatalf("normal termination or SCF convergence unrecognized: %+v",a)}
  if a.Electronic==nil||math.Abs(*a.Electronic-tt.expectedE)>1e-9 {
   t.Fatalf("published real ORCA final energy was parsed incorrectly: expected %.12f; got %v",tt.expectedE,a.Electronic)
  }
  if a.Optimized!=tt.optimized {t.Fatalf("optimization: got %v want %v",a.Optimized,tt.optimized)}
  if !tt.optimized {return}
  if a.ImaginaryCount==nil || *a.ImaginaryCount!=0 {t.Fatalf("real Water minimum has no imaginary modes: got %v",a.ImaginaryCount)}
  if a.Gibbs==nil||math.Abs(*a.Gibbs-(-76.42286885))>1e-7 {t.Fatalf("wrong published Gibbs: %v",a.Gibbs)}
  if a.Enthalpy==nil||math.Abs(*a.Enthalpy-(-76.40144400))>1e-7 {t.Fatalf("wrong published enthalpy: %v",a.Enthalpy)}
  if a.ZPE==nil||math.Abs(*a.ZPE-0.02114567)>1e-7 {t.Fatalf("wrong published ZPE: %v",a.ZPE)}
  if a.Temperature==nil||math.Abs(*a.Temperature-298.15)>1e-2 {t.Fatalf("real ORCA dotted temperature unrecognized: %v",a.Temperature)}
  if len(a.Frequencies)<9 {t.Fatalf("expected >=9 real vibrational frequencies, got %d",len(a.Frequencies))}
  visual:=parseVisualization("water.out",string(out),string(inp))
  if len(visual.Optimization)<2 {t.Fatalf("real Opt should contain several energy cycles: %d",len(visual.Optimization))}
 })}
}
