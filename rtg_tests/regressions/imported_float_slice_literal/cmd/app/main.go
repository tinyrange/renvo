package main

import "example.com/rtgtests/regressions/imported_float_slice_literal/geometry"

func main() {
	bounds := geometry.Bounds{Min: 2, Max: 6}
	values := []geometry.Scalar{bounds.Min, (bounds.Min + bounds.Max) / 2, bounds.Max}
	if len(values) != 3 || values[0] != 2 || values[1] != 4 || values[2] != 6 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
