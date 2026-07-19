package main

func appMain(args []string) int {
	if !(2+3*4 == 14) {
		print("RENVO-0251 multiplication_before_addition failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
