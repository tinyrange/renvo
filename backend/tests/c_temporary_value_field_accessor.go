package main

import "unsafe"

type temporaryValueAccessor struct {
	storage [8]byte
}

func (value *temporaryValueAccessor) __c_ptr_0_value() *uint64 {
	return (*uint64)(unsafe.Pointer(uintptr(unsafe.Pointer(value)) + uintptr(0)))
}

func makeTemporaryValueAccessor(value uint64) temporaryValueAccessor {
	var result temporaryValueAccessor
	*result.__c_ptr_0_value() = value
	return result
}

func appMain(args []string) int {
	if *makeTemporaryValueAccessor(42).__c_ptr_0_value() != 42 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
