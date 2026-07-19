package main

func renvo1020Base() (int, int) {
	return 4, 7
}

func renvo1020Next() (int, int) {
	a, b := renvo1020Base()
	return b, a + b
}

func appMain(args []string) int {
	a, b := renvo1020Next()
	if a != 7 || b != 11 {
		print("RENVO-1020 nested call pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
