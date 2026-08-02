package main

import "unsafe"

func storeWord(address uintptr, index int, value uint32) {
	(*[16]uint32)(unsafe.Pointer(address))[index] = value
}

func storeLocal(address *uint32, index int, value uint32) {
	(*[16]uint32)(unsafe.Pointer(address))[index] = value
}

var useHardware bool

func main() {
	if useHardware {
		storeWord(0x60025098, 3, 42)
	}
	var words [16]uint32
	storeLocal(&words[0], 3, 42)
	if words[3] != 42 {
		print("FAIL: unsafe pointer-to-array index\n")
		return
	}
	print("PASS\n")
}
