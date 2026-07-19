package main

const shape19A = 6
const shape19B = shape19A * 7

var shape19V int = shape19B + 1

func appMain(args []string) int {
	if shape19V != 43 {
		print("program_shape_19 constvar\n")
		return 1
	}
	print("PASS\n")
	return 0
}
