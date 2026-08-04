package main

import "unsafe"

type buffer struct {
	data  [16]byte
	align uint32
}

type descriptor struct {
	control uint32
	buffer  uint32
	next    uint32
}

func address(value *byte) uintptr {
	return uintptr(unsafe.Pointer(value))
}

func main() {
	var value byte
	if address(&value) == 0 {
		print("FAIL: unsafe pointer conversion\n")
		return
	}

	var buffers [2]buffer
	buffers[1].data[7] = 41
	if buffers[1].data[7] != 41 || buffers[0].data[7] != 0 {
		print("FAIL: nested selector assignment\n")
		return
	}

	descriptor := descriptor{control: 17, buffer: 34, next: 51}
	words := (*[3]uint32)(unsafe.Pointer(&descriptor))
	if unsafe.Sizeof(descriptor) != 12 || unsafe.Offsetof(descriptor.buffer) != 4 || unsafe.Offsetof(descriptor.next) != 8 || words[0] != 17 || words[1] != 34 || words[2] != 51 {
		print("FAIL: struct layout\n")
		return
	}
	print("PASS\n")
}
