//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

func configureProcess(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func killTree(pid int)               { _ = syscall.Kill(-pid, syscall.SIGKILL) }
