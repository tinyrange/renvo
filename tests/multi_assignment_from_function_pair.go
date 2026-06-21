package main

func rtg1026Pair() (int, int) {
	return 6, 4
}

func appMain(args []string) int {
	a := 0
	b := 0
	a, b = rtg1026Pair()
	if a-b != 2 {
		print("RTG-1026 assignment from pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
