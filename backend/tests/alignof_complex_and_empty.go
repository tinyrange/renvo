package main

type ComplexField struct{ value complex64 }
type Empty struct{}

func appMain(args []string) int {
	var scalar complex64
	var array [2]complex64
	var field ComplexField
	if Alignof(scalar) != Alignof(float32(0)) || Alignof(array) != Alignof(float32(0)) || Alignof(field) != Alignof(float32(0)) {
		return 1
	}
	if Alignof(Empty{}) != 1 || Alignof([0]int64{}) != Alignof(int64(0)) {
		return 2
	}
	print("PASS\n")
	return 0
}
