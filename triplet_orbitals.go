package main

import (
 "fmt"
 "math"
 "sort"
 "strconv"
 "strings"
)

// UKS occupied-alpha vs unoccupied-beta orbital entries are candidates.
// UHF/UKS alpha/beta orbitals are not necessarily identical in spatial shape.
type tripSOMO struct{
 OrbitalIndex int `json:"orbitalIndex"`
 AlphaOccupation float64 `json:"alphaOccupation"`
 BetaOccupation float64 `json:"betaOccupation"`
 AlphaEnergyEh float64 `json:"alphaEnergyEh"`
}
func tripMethodFingerprint(inp string)string{
 words:=[]string{}
 for _,line:=range strings.Split(inp,"\n"){
  row:=strings.TrimSpace(line)
  if !strings.HasPrefix(row,"!"){continue}
  for _,word:=range strings.Fields(strings.TrimSpace(strings.TrimPrefix(row,"!"))){
   upper:=strings.ToUpper(word)
   switch upper{
   case "RKS","UKS","RHF","UHF","OPT","SP","FREQ","NUMFREQ","ANFREQ","OPTTS","XYZFILE","XYZOUTPUT","PRINTMOS":
    continue
   }
   words=append(words,upper)
  }
 }
 sort.Strings(words)
 return strings.Join(words,"|")
}
func tripSOMOCandidates(raw string)[]tripSOMO{
 type orb struct{index int; occ,eh float64}
 alpha,beta:=map[int]orb{},map[int]orb{}
 channel:=""
 for _,line:=range strings.Split(raw,"\n"){
  u:=strings.ToUpper(strings.TrimSpace(line))
  switch{
  case strings.Contains(u,"ORBITAL ENERGIES"):alpha=map[int]orb{};beta=map[int]orb{};channel=""
  case strings.Contains(u,"SPIN UP ORBITALS"):channel="alpha"
  case strings.Contains(u,"SPIN DOWN ORBITALS"):channel="beta"
  case strings.Contains(u,"MULLIKEN ATOMIC"),strings.Contains(u,"HIRSHFELD"),strings.Contains(u,"MAYER POPULATION"):channel=""
  }
  if channel==""{continue}
  fields:=strings.Fields(line);if len(fields)<4{continue}
  index,err:=strconv.Atoi(fields[0]);if err!=nil{continue}
  occ,e1:=number(fields[1]);energy,e2:=number(fields[2])
  if e1!=nil||e2!=nil||occ<0||occ>2||math.Abs(energy)>1e7{continue}
  r:=orb{index,occ,energy}
  if channel=="alpha"{alpha[index]=r}else{beta[index]=r}
 }
 results:=[]tripSOMO{}
 for _,r:=range alpha{
  b,ok:=beta[r.index]
  if !ok||r.occ<0.85||b.occ>0.15{continue}
  results=append(results,tripSOMO{OrbitalIndex:r.index,AlphaOccupation:r.occ,BetaOccupation:b.occ,AlphaEnergyEh:r.eh})
 }
 sort.Slice(results,func(i,j int)bool{return results[i].OrbitalIndex<results[j].OrbitalIndex})
 return results
}
var _=fmt.Sprintf
