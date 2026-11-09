package main

import (
  "math"
  "os"
  "path/filepath"
  "strings"
  "testing"
)
// Real ORCA 6.1.1 results from independent public test output, downloaded
// only in a temporary CI workspace; proprietary .out files are not committed
// to ORCA Nova, redistributed or included in release artifacts.
// Source: https://github.com/HiroYokoyama/moleditpy_orca_result_analyzer_plugin
// Pinned commit: 5a991e87847391f883e5668299424195dd1bef96
func TestRealORCA611ScanTDDFTAndTriplet(t *testing.T) {
 folder:=os.Getenv("ORCA_REAL_GOLDEN_DIR")
 if folder=="" {t.Skip("ORCA_REAL_GOLDEN_DIR unset")}
 read:=func(t *testing.T,name string)string{t.Helper();b,e:=os.ReadFile(filepath.Join(folder,"public-6.1.1",name));if e!=nil{t.Fatal(e)};return string(b)}
 t.Run("EthaneRelaxedDihedralScan",func(t *testing.T){
  out:=read(t,"ethane-scan.out")
  if !strings.Contains(out,"Program Version 6.1.1"){t.Fatal("wrong ORCA output version")}
  parsed:=parseVisualization("ethane-scan.out",out,"")
  if len(parsed.PES)!=5 {t.Fatalf("5 actual scan steps expected, got %d",len(parsed.PES))}
  expectedCoordinates:=[]float64{-60,-30,0,30,60}
  expectedEnergies:=[]float64{-79.79184680,-79.78974712,-79.78743272,-79.78974733,-79.79184785}
  for i,p:=range parsed.PES{
   if p.Status!="Completed" {t.Fatalf("step %d not converged: %s",i+1,p.Status)}
   if p.CoordinateKind!="dihedral"||p.Coordinate==nil||math.Abs(*p.Coordinate-expectedCoordinates[i])>1e-7{
     t.Fatalf("wrong actual ORCA dihedral step %d: %+v",i+1,p)
   }
   if p.Energy==nil||math.Abs(*p.Energy-expectedEnergies[i])>1e-4{
     t.Fatalf("wrong actual ORCA total energy at scan step %d: %v",i+1,p.Energy)
   }
  }
 })
 t.Run("BenzeneTDDFT",func(t *testing.T){
  out:=read(t,"benzene-opt-vex.out")
  a,e:=ParseOrcaOutput(strings.NewReader(out),"benzene-opt-vex.out","! B3LYP def2-SVP TightSCF\n* xyz 0 1\nC 0 0 0\n*\n")
  if e!=nil{t.Fatal(e)}
  if len(a.Roots)<5{t.Fatalf("expected 5 real TD-DFT root energies; got %d",len(a.Roots))}
  expected:=[]float64{5.494,6.469,7.485,7.485,7.715}
  for i,want:=range expected {
    r:=a.Roots[i]
    if r.Kind!="S"||r.Root!=i+1||math.Abs(r.EV-want)>0.005{t.Fatalf("root %d: %+v",i+1,r)}
  }
 })
 t.Run("O2TripletSpin",func(t *testing.T){
  out:=read(t,"o2-opt_orb.out")
  a,e:=ParseOrcaOutput(strings.NewReader(out),"o2-opt_orb.out","! PBE0 def2-SVP Opt Freq\n* xyz 0 3\nO 0 0 0\nO 1 0 0\n*\n")
  if e!=nil{t.Fatal(e)}
  if a.S2==nil||math.Abs(*a.S2-2.007028)>0.02{t.Fatalf("O2 triplet <S^2> not parsed: %v",a.S2)}
  pop:=a.Populations["Mulliken"]
  if len(pop)!=2 {t.Fatalf("two oxygen populations expected: %+v",pop)}
  for _,atom:=range pop {
   if atom.Spin==nil||math.Abs(*atom.Spin-1.0)>0.01{t.Fatalf("O2 atom expected spin ~1 electron: %+v",atom)}
  }
 })
}
