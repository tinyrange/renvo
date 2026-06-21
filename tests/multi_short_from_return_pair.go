package main

func rtg1039Pair() (int, int) {
	return 11, 13
}

func appMain(args []string) int {
	a, b := rtg1039Pair()
	if a+b != 24 {
		print("RTG-1039 short from return pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
