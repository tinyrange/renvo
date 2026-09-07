package main

func appMain(args []string) int {
	imag := func(s string) int { return len(s) }
	if imag("abc") != 3 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
