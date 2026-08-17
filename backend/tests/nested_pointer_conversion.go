package main

import "unsafe"

func appMain(args []string) int {
	value := 42
	pointer := &value
	converted := (**int)(unsafe.Pointer(&pointer))
	if **converted != 42 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
