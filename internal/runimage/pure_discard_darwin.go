//go:build !renvo && darwin && (amd64 || arm64)

package runimage

import "syscall"

func pureDiscard(base uintptr, size int) error {
	_, _, err := syscall.Syscall(syscall.SYS_MADVISE, base, uintptr(size), syscall.MADV_DONTNEED)
	if err != 0 {
		return err
	}
	return nil
}
