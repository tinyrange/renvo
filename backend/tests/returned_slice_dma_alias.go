package main

import "unsafe"

func alignedBytes(n int) ([]byte, uintptr) {
	b := make([]byte, n+63)
	a := uintptr(unsafe.Pointer(&b[0]))
	offset := int((64 - a%64) % 64)
	return b[offset : offset+n], a + uintptr(offset)
}

func appMain() int {
	b, address := alignedBytes(512)
	if uintptr(unsafe.Pointer(&b[0])) != address {
		print("FAIL alias\n")
		return 1
	}
	*(*byte)(unsafe.Pointer(address)) = 42
	if b[0] != 42 {
		print("FAIL write\n")
		return 1
	}
	print("PASS\n")
	return 0
}
