package main

import (
 "fmt"
 "regexp"
 "runtime"
 "strings"
)

type resourceSystem struct {
 CPU int `json:"cpu"`
 TotalMB int `json:"totalMB"`
 AvailableMB int `json:"availableMB"`
 BudgetMB int `json:"budgetMB"`
 Supported bool `json:"supported"`
 Message string `json:"message,omitempty"`
}

func memoryBudget(total, available int) int {
 if total <= 0 || available <= 0 || available > total { return 0 }
 // Leave 20% of currently available physical memory, with at least 1 GiB reserved.
 return max(0, min(available*4/5, available-1024))
}

func systemResources() resourceSystem {
 s := resourceSystem{CPU: runtime.NumCPU()}
 total, available, err := physicalMemoryMB()
 if err != nil { s.Message = err.Error(); return s }
 s.TotalMB, s.AvailableMB = total, available
 s.BudgetMB = memoryBudget(total, available)
 s.Supported = total > 0 && available > 0 && available <= total
 return s
}

// Only canonical application-generated resource directives may be rewritten.
// Imported and edited input files are not eligible for automatic rewriting.
var autoPALBlock = regexp.MustCompile(`(?im)^%pal[ \t]*\r?\n[ \t]+nprocs[ \t]+[0-9]+[ \t]*\r?\nend[ \t]*\r?$`)
var autoMaxCoreLine = regexp.MustCompile(`(?im)^%maxcore[ \t]+[0-9]+[ \t]*\r?$`)

func autoRewrite(req RunRequest, cores, maxcore int) (RunRequest, error) {
 if _, _, err := resources(req); err != nil { return req, err }
 updated := req
 updated.Files = append([]JobFile(nil), req.Files...)
 for _, st := range req.Stages {
  found := false
  for i := range updated.Files {
   f := &updated.Files[i]
   if f.Name != st.Input { continue }
   found = true
   if len(autoPALBlock.FindAllStringIndex(f.Content,-1)) != 1 || len(autoMaxCoreLine.FindAllStringIndex(f.Content,-1)) != 1 {
    return req, fmt.Errorf("%s: 자동 배분은 앱에서 생성한 표준 PAL/MaxCore만 지원합니다. Manual 모드를 사용하세요", st.Input)
   }
   nl := "\n"
   if strings.Contains(f.Content,"\r\n") { nl = "\r\n" }
   f.Content = autoPALBlock.ReplaceAllStringFunc(f.Content,func(v string)string{
    ending := ""; if strings.HasSuffix(v,"\r") {ending="\r"}
    return fmt.Sprintf("%%pal%s  nprocs %d%send%s", nl,cores,nl,ending)
   })
   f.Content = autoMaxCoreLine.ReplaceAllStringFunc(f.Content,func(v string)string{
    ending := ""; if strings.HasSuffix(v,"\r") {ending="\r"}
    return fmt.Sprintf("%%maxcore %d%s",maxcore,ending)
   })
  }
  if !found {return req, fmt.Errorf("단계 INP를 찾을 수 없습니다: %s",st.Input)}
 }
 return updated,nil
}

type allocationView struct {
 Label string `json:"label"`
 CPU int `json:"cpu"`
 RAMMB int `json:"ramMB"`
 Automatic bool `json:"automatic"`
}

const autoMinimumMaxCore = 512
const autoParallelCap = 4

// Plan using a fresh memory snapshot, never trusting a client-provided RAM budget.
// Fixed inputs keep their original reservations, and the existing scheduler
// enforces the sum of running reservations.
func planAutoResources(jobs []RunRequest, flags []bool, cpu int, sys resourceSystem) ([]RunRequest, QueueLimits, error) {
 if len(jobs)==0 || len(jobs)!=len(flags) { return nil,QueueLimits{},fmt.Errorf("자동 배분 대상 목록이 올바르지 않습니다") }
 if !sys.Supported || sys.BudgetMB>1000000000 || cpu<1 || cpu>sys.CPU { return nil,QueueLimits{},fmt.Errorf("시스템 자원 감지 또는 CPU 선택을 확인하세요. Manual 모드도 사용할 수 있습니다") }
 cores:=min(cpu,sys.BudgetMB/(autoMinimumMaxCore*5/4))
 if cores<1 {return nil,QueueLimits{},fmt.Errorf("사용 가능한 RAM이 부족합니다. 다른 프로그램을 종료한 후 다시 시도하세요")}
 parallel:=min(len(jobs),cores,autoParallelCap)
 perCore:=min(16000,((sys.BudgetMB*4/5)/cores)/4*4)
 limits:=QueueLimits{Concurrency:parallel,CPU:cpu,RAMMB:sys.BudgetMB}
 out:=append([]RunRequest(nil),jobs...)
 for i, req:=range jobs{
  if !flags[i] {continue}
  slot:=i%parallel
  n:=cores/parallel
  if slot<cores%parallel {n++}
  rewritten,err:=autoRewrite(req,n,perCore)
  if err!=nil{return nil,QueueLimits{},err}
  out[i]=rewritten
 }
 return out,limits,nil
}
