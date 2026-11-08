package main

import (
	"fmt"
	"os/exec"
)

func configureProcess(cmd *exec.Cmd) {}
func killTree(pid int)               { _ = exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprint(pid)).Run() }
