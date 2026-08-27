//go:build !renvo || !uefi || !amd64

package main

import "unsafe"

func zeroMemory(address, size uintptr) {
	for i := uintptr(0); i < size; i++ {
		*(*byte)(unsafe.Pointer(address + i)) = 0
	}
}

func enterLinux64(entry, bootParams, stackTop uintptr) {
	panic("UEFI Linux entry is unavailable on the host")
}
