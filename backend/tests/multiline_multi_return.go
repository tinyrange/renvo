package main

func multilineValues() (int, int, int, int) {
	return 1,
		2,
		3,
		4
}

func appMain(args []string) int {
	a, b, c, d := multilineValues()
	if a+b+c+d != 10 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
