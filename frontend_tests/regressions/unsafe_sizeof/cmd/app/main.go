package main

import "unsafe"

type sample struct {
	Value int
	Flag  bool
}

func main() {
	var value int
	if unsafe.Sizeof(value) == 8 && unsafe.Sizeof(&value) == 8 && unsafe.Sizeof([2]byte{}) == 2 && unsafe.Sizeof(sample{}) == 16 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
