package main

func rtg1020Base() (int, int) {
	return 4, 7
}

func rtg1020Next() (int, int) {
	a, b := rtg1020Base()
	return b, a + b
}

func appMain(args []string) int {
	a, b := rtg1020Next()
	if a != 7 || b != 11 {
		print("RTG-1020 nested call pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
