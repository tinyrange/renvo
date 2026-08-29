package main

import "unsafe"

type sample struct {
	Value int
	Flag  bool
}

func main() {
	var value int
	if unsafe.Sizeof(value) == expectedIntSize && unsafe.Sizeof(&value) == expectedPointerSize && unsafe.Sizeof([2]byte{}) == 2 && unsafe.Sizeof(sample{}) == expectedSampleSize {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
