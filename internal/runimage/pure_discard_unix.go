//go:build !renvo && linux && (amd64 || arm64)

package runimage

import (
	"syscall"
	"unsafe"
)

func pureDiscard(base uintptr, size int) error {
	return syscall.Madvise(unsafe.Slice((*byte)(unsafe.Pointer(base)), size), syscall.MADV_DONTNEED)
}
