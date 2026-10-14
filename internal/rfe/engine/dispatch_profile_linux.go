//go:build !renvo && linux

package engine

import (
	"syscall"
	"unsafe"
)

func profileMonotonicNS() uint64 {
	var ts syscall.Timespec
	_, _, err := syscall.Syscall(syscall.SYS_CLOCK_GETTIME, 1, uintptr(unsafe.Pointer(&ts)), 0)
	if err != 0 {
		return 0
	}
	return uint64(ts.Sec)*1000000000 + uint64(ts.Nsec)
}
