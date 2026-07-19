package main

func appMain(args []string) int {
	a := 0
	b := 0
	if len(args) > 0 {
		a, b = 6, 7
	} else {
		a, b = 1, 2
	}
	if a*b != 42 {
		print("RENVO-1034 branch assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
