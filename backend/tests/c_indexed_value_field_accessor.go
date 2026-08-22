package main

import "unsafe"

type indexedValueAccessorEntry struct {
	storage [16]byte
}

func (entry *indexedValueAccessorEntry) __c_ptr_0_value() *uint64 {
	return (*uint64)(unsafe.Pointer(uintptr(unsafe.Pointer(entry)) + uintptr(0)))
}

func appMain(args []string) int {
	var entries [2]indexedValueAccessorEntry
	index := 1
	*entries[index].__c_ptr_0_value() = 42
	if *entries[index].__c_ptr_0_value() != 42 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
