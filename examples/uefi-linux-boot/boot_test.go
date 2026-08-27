package main

import (
	"testing"
	"unsafe"
)

func TestPrepareIdentityMap(t *testing.T) {
	memory := make([]byte, 7*4096)
	root := uintptr(unsafe.Pointer(&memory[0]))
	prepareIdentityMap(root)

	if got, want := load64(root, 0), uint64(root+4096)|3; got != want {
		t.Fatalf("PML4[0] = %#x, want %#x", got, want)
	}
	for gigabyte := uintptr(0); gigabyte < 4; gigabyte++ {
		directory := root + (2+gigabyte)*4096
		if got, want := load64(root+4096, gigabyte*8), uint64(directory)|3; got != want {
			t.Fatalf("PDPT[%d] = %#x, want %#x", gigabyte, got, want)
		}
		for _, page := range []uintptr{0, 1, 511} {
			physical := (gigabyte << 30) + (page << 21)
			if got, want := load64(directory, page*8), uint64(physical)|0x83; got != want {
				t.Fatalf("directory %d page %d = %#x, want %#x", gigabyte, page, got, want)
			}
		}
	}
	if got, want := load64(root+6*4096, 0), uint64(root)|3; got != want {
		t.Fatalf("PML5[0] = %#x, want %#x", got, want)
	}
}

func TestPrepareLinuxTransition(t *testing.T) {
	memory := make([]byte, 4096)
	page := uintptr(unsafe.Pointer(&memory[0]))
	trampoline := prepareLinuxTransition(page)
	if trampoline != page+64 {
		t.Fatalf("trampoline = %#x, want %#x", trampoline, page+64)
	}
	want := []byte{0x0f, 0x20, 0xe0, 0xa9, 0x00, 0x10, 0x00, 0x00, 0x74, 0x03,
		0x4c, 0x89, 0xd7, 0x0f, 0x22, 0xdf, 0x31, 0xed, 0x31, 0xdb, 0x31, 0xff,
		0x41, 0xff, 0xe0}
	for i := 0; i < len(want); i++ {
		if got := *(*byte)(unsafe.Pointer(trampoline + uintptr(i))); got != want[i] {
			t.Fatalf("trampoline byte %d = %#x, want %#x", i, got, want[i])
		}
	}
}
