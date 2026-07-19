package main

func appMain(args []string) int {
	if !(((2 + 3) * (4 - 1)) == 15) {
		print("RENVO-0821 parentheses failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
