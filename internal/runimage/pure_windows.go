//go:build !renvo && windows && (amd64 || arm64)

package runimage

import (
	"fmt"
	"syscall"
	"unsafe"
)

var pureKernel = syscall.NewLazyDLL("kernel32.dll")
var pureAlloc = pureKernel.NewProc("VirtualAlloc")
var pureProtect = pureKernel.NewProc("VirtualProtect")
var pureFree = pureKernel.NewProc("VirtualFree")
var pureCache = pureKernel.NewProc("FlushInstructionCache")

func pureMap(size int) (uintptr, error) {
	p, _, err := pureAlloc.Call(0, uintptr(size), 0x3000, 4)
	if p == 0 {
		return 0, fmt.Errorf("VirtualAlloc: %w", err)
	}
	return p, nil
}
func pureWritable(base uintptr, size int, write bool) error {
	protection := uintptr(0x20)
	if write {
		protection = 4
	}
	var old uint32
	r, _, err := pureProtect.Call(base, uintptr(size), protection, uintptr(unsafe.Pointer(&old)))
	if r == 0 {
		return err
	}
	return nil
}
func pureFlush(base uintptr, size int) { pureCache.Call(^uintptr(0), base, uintptr(size)) }
func pureUnmap(base uintptr, size int) error {
	r, _, err := pureFree.Call(base, 0, 0x8000)
	if r == 0 {
		return err
	}
	return nil
}
