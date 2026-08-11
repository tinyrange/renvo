//go:build !linux

package main

import "os"

func processMaxRSSKB(state *os.ProcessState) int {
	return 0
}
