package main

func renvo1001Pair() (int, int) {
	return 4, 9
}

func appMain(args []string) int {
	a, b := renvo1001Pair()
	if a != 4 || b != 9 || a+b != 13 {
		print("RENVO-1001 two int returns failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
