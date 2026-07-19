package main

func appMain(args []string) int {
	if !((1+2)<<3 == 24) {
		print("RENVO-0239 shift_after_addition_with_parentheses failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
