package main

var renvo0503Start int = 6

func renvo0503Sum(n int) int {
	if n == 0 {
		return 0
	}
	return n + renvo0503Sum(n-1)
}

func appMain(args []string) int {
	if renvo0503Sum(renvo0503Start) != 21 {
		print("RENVO-0503 sum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
