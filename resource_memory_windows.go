package main

import (
 "fmt"
 "syscall"
 "unsafe"
)

type memoryStatusEx struct {
 Length uint32
 MemoryLoad uint32
 TotalPhys uint64
 AvailPhys uint64
 TotalPageFile uint64
 AvailPageFile uint64
 TotalVirtual uint64
 AvailVirtual uint64
 AvailExtendedVirtual uint64
}

var memoryStatusProc = syscall.NewLazyDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")

func physicalMemoryMB() (total, available int, err error) {
 var s memoryStatusEx
 s.Length = uint32(unsafe.Sizeof(s))
 ok, _, e := memoryStatusProc.Call(uintptr(unsafe.Pointer(&s)))
 if ok == 0 { return 0, 0, fmt.Errorf("GlobalMemoryStatusEx: %v", e) }
 return int(s.TotalPhys >> 20), int(s.AvailPhys >> 20), nil
}

var systemTimesProc = syscall.NewLazyDLL("kernel32.dll").NewProc("GetSystemTimes")

func physicalCPUTimes() (cpuCounters,bool) {
 var idle,kernel,user syscall.Filetime
 ok,_,_:=systemTimesProc.Call(uintptr(unsafe.Pointer(&idle)),uintptr(unsafe.Pointer(&kernel)),uintptr(unsafe.Pointer(&user)))
 ticks:=func(t syscall.Filetime)uint64{return uint64(t.HighDateTime)<<32|uint64(t.LowDateTime)}
 if ok==0{return cpuCounters{},false}
 return cpuCounters{idle:ticks(idle),kernel:ticks(kernel),user:ticks(user)},true
}
