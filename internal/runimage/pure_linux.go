//go:build !renvo && linux && (amd64 || arm64)

package runimage

import (
	"syscall"
	"unsafe"
)

func pureMap(size int) (uintptr, error) {
	b, err := syscall.Mmap(-1, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		return 0, err
	}
	return uintptr(unsafe.Pointer(&b[0])), nil
}
func pureWritable(base uintptr, size int, write bool) error {
	prot := syscall.PROT_READ | syscall.PROT_EXEC
	if write {
		prot = syscall.PROT_READ | syscall.PROT_WRITE
	}
	return syscall.Mprotect(unsafe.Slice((*byte)(unsafe.Pointer(base)), size), prot)
}
func pureUnmap(base uintptr, size int) error {
	return syscall.Munmap(unsafe.Slice((*byte)(unsafe.Pointer(base)), size))
}
