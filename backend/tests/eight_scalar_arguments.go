package main

func eightScalarArguments(a, b, c, d, e, f, g, h int) int {
	return a + 2*b + 3*c + 4*d + 5*e + 6*f + 7*g + 8*h
}

func appMain() int {
	if eightScalarArguments(1, 2, 3, 4, 5, 6, 7, 8) != 204 {
		print("FAIL overflow arguments\n")
		return 1
	}
	print("PASS\n")
	return 0
}
