package main

var C, D int

func appMain(args []string) int {
	C = 3
	D = 4
	if C+D == 7 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
