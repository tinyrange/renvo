//go:build !renvo && windows && (amd64 || arm64)

package runimage

import (
	"fmt"
	"unsafe"
)

func pureMap(size int) (uintptr, error) {
	p, _, err := windowsVirtualAlloc.Call(0, uintptr(size), windowsMemCommit|windowsMemReserve, windowsPageReadWrite)
	if p == 0 {
		return 0, fmt.Errorf("VirtualAlloc: %w", err)
	}
	return p, nil
}
func pureWritable(base uintptr, size int, write bool) error {
	protection := uintptr(windowsPageExecuteRead)
	if write {
		protection = windowsPageReadWrite
	}
	var old uint32
	r, _, err := windowsVirtualProtect.Call(base, uintptr(size), protection, uintptr(unsafe.Pointer(&old)))
	if r == 0 {
		return err
	}
	return nil
}
func pureSeal(base uintptr, size, at, count int) error {
	if err := pureWritable(base, size, false); err != nil {
		return err
	}
	r, _, err := windowsFlushInstructionCache.Call(^uintptr(0), base+uintptr(at), uintptr(count))
	if r == 0 {
		return windowsCallError("FlushInstructionCache", err)
	}
	return nil
}
func pureUnmap(base uintptr, size int) error {
	r, _, err := windowsVirtualFree.Call(base, 0, windowsMemRelease)
	if r == 0 {
		return err
	}
	return nil
}
