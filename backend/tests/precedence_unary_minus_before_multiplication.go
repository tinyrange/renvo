package main

func appMain(args []string) int {
	if !(-3*4 == -12) {
		print("RENVO-0260 unary_minus_before_multiplication failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
