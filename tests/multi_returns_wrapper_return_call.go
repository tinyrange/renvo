package main

func rtg1010Inner() (int, int) {
	return 14, 3
}

func rtg1010Outer() (int, int) {
	return rtg1010Inner()
}

func appMain(args []string) int {
	a, b := rtg1010Outer()
	if a-b != 11 {
		print("RTG-1010 wrapper return call failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
