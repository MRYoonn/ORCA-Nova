package main

import (
 "runtime"
 "sync"
 "time"
)

type resourceUsage struct {
 CPUPercent *float64 `json:"cpuPercent"`
 RAMUsedMB *int `json:"ramUsedMB"`
 RAMTotalMB *int `json:"ramTotalMB"`
 RAMAvailableMB *int `json:"ramAvailableMB"`
}
type cpuCounters struct{ idle,kernel,user uint64 }

// Windows kernel time includes idle time. Discard invalid or reset counters.
func cpuUsageDelta(before,after cpuCounters) (float64,bool) {
 if after.idle<before.idle||after.kernel<before.kernel||after.user<before.user {return 0,false}
 idle:=after.idle-before.idle
 kernel:=after.kernel-before.kernel
 user:=after.user-before.user
 total:=kernel+user
 if total==0||idle>total||total<kernel{return 0,false}
 return 100*float64(total-idle)/float64(total),true
}
var usageSampler struct{
 sync.Mutex
 last cpuCounters
 stamp time.Time
 percent *float64
}

func systemUsage() resourceUsage {
 usage:=resourceUsage{}
 if total,available,err:=physicalMemoryMB();err==nil&&total>0&&available>=0&&available<=total{
  used:=total-available
  usage.RAMUsedMB=&used;usage.RAMTotalMB=&total;usage.RAMAvailableMB=&available
 }
 // GetSystemTimes reports the calling processor group on hosts above 64 CPUs.
 // Do not present a processor-group-only value as the whole machine's usage.
 if runtime.NumCPU()>64{return usage}
 usageSampler.Lock()
 defer usageSampler.Unlock()
 now:=time.Now()
 if !usageSampler.stamp.IsZero()&&now.Sub(usageSampler.stamp)<500*time.Millisecond{usage.CPUPercent=usageSampler.percent;return usage}
 sample,ok:=physicalCPUTimes()
 if !ok{usageSampler.stamp=time.Time{};usageSampler.percent=nil;return usage}
 if !usageSampler.stamp.IsZero(){
  if percent,valid:=cpuUsageDelta(usageSampler.last,sample);valid{usageSampler.percent=&percent}else{usageSampler.percent=nil}
 }
 usageSampler.last=sample;usageSampler.stamp=now
 usage.CPUPercent=usageSampler.percent
 return usage
}
