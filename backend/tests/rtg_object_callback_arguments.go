package main

import "unsafe"

//export ProbeEightArguments
func probeEightArguments(a, b, c, d, e, f, g, h uintptr) uintptr {
	return a + 3*b + 5*c + 7*d + 11*e + 13*f + 17*g + 19*h
}

func verifyObjectCallback() bool {
	callback := probeEightArguments
	address := *(*uintptr)(unsafe.Pointer(&callback))
	var restored func(uintptr, uintptr, uintptr, uintptr, uintptr, uintptr, uintptr, uintptr) uintptr
	*(*uintptr)(unsafe.Pointer(&restored)) = address
	return restored(2, 3, 5, 7, 11, 13, 17, 19) == 1025
}

func appMain() int {
	if !verifyObjectCallback() {
		return 1
	}
	print("PASS\n")
	return 0
}
