//go:build !renvo && linux && !amd64

package runimage

import (
	"syscall"
	"unsafe"
)

const jitTimestampFlags = 0

func jitTimestamp() (uint64, error) {
	var ts syscall.Timespec
	_, _, err := syscall.Syscall(syscall.SYS_CLOCK_GETTIME, 1, uintptr(unsafe.Pointer(&ts)), 0)
	if err != 0 {
		return 0, err
	}
	return uint64(ts.Sec)*1000000000 + uint64(ts.Nsec), nil
}
