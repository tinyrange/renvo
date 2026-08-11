//go:build linux

package main

import (
	"os"
	"syscall"
)

func processMaxRSSKB(state *os.ProcessState) int {
	usage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok {
		return 0
	}
	return int(usage.Maxrss)
}
