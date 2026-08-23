package main

func ieeeComplex64Identity(value complex64) complex64 { return value }

func appMain(args []string) int {
	left := ieeeComplex64Identity(complex(float32(1), float32(2)))
	same := ieeeComplex64Identity(complex(float32(1), float32(2)))
	different := ieeeComplex64Identity(complex(float32(1), float32(3)))
	if left != same || left == different {
		print("FAIL: complex64 component comparison\n")
		return 1
	}

	zero := float32(0)
	negativeZero := -zero
	if complex(zero, negativeZero) != complex(negativeZero, zero) {
		print("FAIL: complex64 signed zero comparison\n")
		return 1
	}

	nan := zero / zero
	unordered := ieeeComplex64Identity(complex(nan, zero))
	if unordered == unordered || !(unordered != unordered) {
		print("FAIL: complex64 unordered comparison\n")
		return 1
	}

	var boxedLeft interface{} = left
	var boxedSame interface{} = same
	var boxedDifferent interface{} = different
	if boxedLeft != boxedSame || boxedLeft == boxedDifferent {
		print("FAIL: interface complex64 comparison\n")
		return 1
	}

	print("PASS\n")
	return 0
}
