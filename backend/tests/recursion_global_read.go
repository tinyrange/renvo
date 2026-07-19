package main

var renvo0516Limit int = 4

func renvo0516Product(n int) int {
	if n > renvo0516Limit {
		return 1
	}
	return n * renvo0516Product(n+1)
}

func appMain(args []string) int {
	var checks []int
	checks = append(checks, renvo0516Product(1))
	if checks[0] != 24 {
		print("RENVO-0516 global read failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
