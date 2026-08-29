package main

// renvo:linkstatic /usr/lib/libSystem.B.dylib,abs,float64=3
func darwinMixedMany(a, b, c, d, e, f, g, h, i, j int) int { return 3 }

func appMain() int {
	// The float mask sends the first two words through FP registers, leaving
	// the third argument as abs's first integer argument in x0.
	if darwinMixedMany(1, 2, -3, 4, 5, 6, 7, 8, 9, 10) != 3 {
		return 1
	}
	print("PASS\n")
	return 0
}
