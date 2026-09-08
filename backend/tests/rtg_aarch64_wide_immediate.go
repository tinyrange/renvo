package main

import "unsafe"

func checkWideImmediate() bool {
	x := uint64(0x51454d5520434647)
	b := (*[8]byte)(unsafe.Pointer(&x))
	return b[0] == 0x47 && b[1] == 0x46 && b[2] == 0x43 && b[3] == 0x20 &&
		b[4] == 0x55 && b[5] == 0x4d && b[6] == 0x45 && b[7] == 0x51
}

func appMain() int {
	if !checkWideImmediate() {
		return 1
	}
	print("PASS\n")
	return 0
}
