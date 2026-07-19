package main

import "example.com/renvotests/regressions/typed_float_constant_arithmetic/geometry"

const rowHeight geometry.Scalar = 34

func main() {
	y := geometry.Scalar(5)
	y += rowHeight
	if y != 39 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
