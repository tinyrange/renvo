package main

func appMain(args []string) int {
	if !(4 == 4) {
		print("RENVO-0176 integer_equality_true failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
