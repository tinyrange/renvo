package main

// renvo:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend,float64=3
func darwinMixedMany(a, b, c, d, e, f, g, h, i, j int) int { return 7 }

func appMain() int {
	if darwinMixedMany(1, 2, 3, 4, 5, 6, 7, 8, 9, 10) != 7 {
		return 1
	}
	print("PASS\n")
	return 0
}
