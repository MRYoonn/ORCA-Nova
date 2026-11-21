//go:build !windows

package main

import "fmt"

// The automatic policy is not enabled without a supported live memory probe.
func physicalMemoryMB() (int, int, error) {
 return 0, 0, fmt.Errorf("자동 RAM 감지는 Windows에서 지원됩니다")
}

func physicalCPUTimes() (cpuCounters,bool) {return cpuCounters{},false}
