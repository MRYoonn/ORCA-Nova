package main

import (
 "strings"
 "testing"
)
func TestORCADottedTemperatureInThermochemistry(t *testing.T) {
 input:="! PBE0 def2-SVP TightSCF Opt Freq\n* xyzfile 0 1 water.xyz"
 out:=strings.Join([]string{
 "SCF CONVERGED",
 "THE OPTIMIZATION HAS CONVERGED",
 "FINAL SINGLE POINT ENERGY -151.555",
 "VIBRATIONAL FREQUENCIES",
 "  0:  800.0000 cm**-1",
 "Temperature                     ...   298.15 K",
 "Final Gibbs free energy         ...   -151.500000 Eh",
 "ORCA TERMINATED NORMALLY",
 }, "\n")
 a,e:=ParseOrcaOutput(strings.NewReader(out),"water.out",input);if e!=nil{t.Fatal(e)}
 if a.Temperature==nil || *a.Temperature!=298.15 {t.Fatalf("ORCA dotted Temperature not recognized: %+v",a.Temperature)}
 if a.Gibbs==nil || *a.Gibbs!=-151.5 {t.Fatalf("ORCA dotted Gibbs not recognized: %+v",a.Gibbs)}
}
