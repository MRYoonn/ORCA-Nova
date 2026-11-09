package main

import (
 "fmt"
 "io"
 "math"
 "sort"
 "strings"
)

// exportHumanTextLocked provides an offline, molecule-by-molecule research
// overview. The caller holds q.mu and the parser reads only that job's OUT.
// No cross-molecule reaction ΔG is inferred from absolute energies.
func (q *JobQueue) exportHumanTextLocked(out io.Writer) {
 fmt.Fprintln(out,"ORCA Nova | Batch Research Summary")
 fmt.Fprintln(out,"Electronic/Gibbs/Enthalpy: absolute Eh; ZPE: correction Eh")
 fmt.Fprintln(out,"Vertical S1/T1: kcal/mol (eV) at the OUT's original geometry")
 fmt.Fprintln(out,"Not calculated is never substituted with invented values.")
 fmt.Fprintln(out,"Warning: Mock CI results are not a substitute for actual ORCA 6.x validation.")
 fmt.Fprintln(out,"C5/C6 cannot be inferred from zero-based ORCA atom indices.")
 fmt.Fprintln(out,"")
 for _,job:=range q.Jobs {
  fmt.Fprintf(out,"Molecule: %s | Queue: %s | CPU: %d | RAM reservation: %d MB\n",
   job.Label,job.State,job.CPU,job.RAMMB)
  if job.Job!=nil {
   fmt.Fprintf(out,"Folder: %s\nMessage: %s\n",job.Job.Folder,job.Job.Message)
   fmt.Fprintln(out,"Per-job detailed interpretation: analysis.txt / results.json")
  }
  for _,stage:=range job.Request.Stages {
   fmt.Fprintf(out,"  Stage: %s (%s)\n",stage.Name,stage.Input)
   a,ok:=batchStageAnalysis(job,stage)
   if !ok {
    fmt.Fprintln(out,"    Status: Not calculated / OUT unavailable or unreadable")
    continue
   }
   fmt.Fprintf(out,"    Status: %s | normal: %t | SCF: %t | Opt converged: %t\n",
    a.Status,a.Normal,a.SCFConverged,a.Optimized)
   fmt.Fprintf(out,"    Method: %s | Geometry: %s | Charge: %d | Multiplicity: %d\n",
    a.Method,a.Geometry,a.Charge,a.Multiplicity)
   if a.ImaginaryCount!=nil {fmt.Fprintf(out,"    Imaginary frequencies: %d\n",*a.ImaginaryCount)}
   if a.S2!=nil {fmt.Fprintf(out,"    <S^2>: %s\n",batchEnergyCSV(a.S2))}
   if a.Normal&&a.SCFConverged&&(!stage.Opt||a.Optimized) {
    fmt.Fprintf(out,"    Electronic: %s Eh | Gibbs: %s Eh | Enthalpy: %s Eh | ZPE correction: %s Eh\n",
     batchEnergyCSV(a.Electronic),batchEnergyCSV(a.Gibbs),
     batchEnergyCSV(a.Enthalpy),batchEnergyCSV(a.ZPE))
    for _,kind:=range []struct{label,letter string}{{"S1","s"},{"T1","t"}} {
     kcal,eV:=batchRootCSV(a.Roots,kind.letter)
     fmt.Fprintf(out,"    Vertical %s: %s kcal/mol (%s eV)\n",kind.label,kcal,eV)
    }
    for _,scheme:=range []string{"Mulliken","Loewdin","Hirshfeld"} {
     spins:=append([]AtomicPopulation(nil),a.Populations[scheme]...)
     sort.SliceStable(spins,func(i,j int)bool{
      if spins[i].Spin==nil{return false}
      if spins[j].Spin==nil{return true}
      return math.Abs(*spins[i].Spin)>math.Abs(*spins[j].Spin)
     })
     shown:=0
     for _,atom:=range spins {
      if atom.Spin==nil||!isFinite(*atom.Spin) {continue}
      if shown>=8 {break}
      fmt.Fprintf(out,"    %s spin: ORCA %d %s = %+0.4f\n",
       scheme,atom.Index,atom.Element,*atom.Spin)
      shown++
     }
    }
   } else {
    fmt.Fprintln(out,"    Numeric results suppressed: calculation incomplete or unconverged.")
   }
   if len(a.Warnings)>0 {
    fmt.Fprintf(out,"    Warnings: %s\n",strings.Join(a.Warnings," | "))
   }
  }
  fmt.Fprintln(out,"")
 }
 fmt.Fprintln(out,"Adiabatic S0/T1, reaction ΔE/ΔG/ΔG‡ require explicit state/stoichiometry/method/geometry verification.")
 fmt.Fprintln(out,"Use the verified Energy Profile UI and each job's analysis.txt for such comparisons.")
}
