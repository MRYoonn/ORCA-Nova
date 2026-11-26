package main

import (
 "strings"
 "testing"
)

func TestM062XKeywordPaths(t *testing.T) {
 for _, v := range []string{"M06-2X","m06-2x","M062X"} {
  if got:=orcaFunctionalKeyword(v);got!="M062X" {t.Fatalf("%s: %s",v,got)}
 }
 if got:=orcaFunctionalKeyword("B3LYP");got!="B3LYP" {t.Fatal(got)}
 opt:=tripOptions{Functional:"M06-2X",Basis:"def2-TZVP",Dispersion:"D4",Solvent:"acetonitrile",Cores:8,MaxCore:2000}
 for _, stage:=range []string{"constrained","free","spin"} {
  k:=keywordLine(tripORCAInput(stage,"input.xyz",opt,nil))
  if !k["m062x"]||k["m06-2x"] {t.Fatalf("%s: wrong keyword %v",stage,k)}
 }
 req:=hatTestRequest()
 for i:=range req.Files {
  if strings.HasSuffix(req.Files[i].Name,".inp") {req.Files[i].Content=strings.ReplaceAll(req.Files[i].Content,"M062X","M06-2X")}
 }
 r:=preflight(req)
 if r.OK||!strings.Contains(strings.Join(r.Errors," "),"M062X") {t.Fatalf("legacy invalid keyword not diagnosed: %+v",r)}
}
