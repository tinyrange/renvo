package main

func renvo1026Pair() (int, int) {
	return 6, 4
}

func appMain(args []string) int {
	a := 0
	b := 0
	a, b = renvo1026Pair()
	if a-b != 2 {
		print("RENVO-1026 assignment from pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
