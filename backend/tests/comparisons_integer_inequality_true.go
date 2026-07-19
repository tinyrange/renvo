package main

func appMain(args []string) int {
	if !(4 != 5) {
		print("RENVO-0178 integer_inequality_true failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
