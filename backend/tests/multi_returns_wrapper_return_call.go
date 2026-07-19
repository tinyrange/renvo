package main

func renvo1010Inner() (int, int) {
	return 14, 3
}

func renvo1010Outer() (int, int) {
	return renvo1010Inner()
}

func appMain(args []string) int {
	a, b := renvo1010Outer()
	if a-b != 11 {
		print("RENVO-1010 wrapper return call failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
