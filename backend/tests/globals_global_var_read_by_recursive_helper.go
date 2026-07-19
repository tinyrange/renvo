package main

var renvo0695Step int = 3

func renvo0695Sum(n int) int {
	if n == 0 {
		return 0
	}
	return renvo0695Step + renvo0695Sum(n-1)
}

func appMain(args []string) int {
	if renvo0695Sum(4) != 12 {
		print("RENVO-0695 recursive global read failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
