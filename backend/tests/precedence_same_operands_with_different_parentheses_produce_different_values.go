package main

func appMain(args []string) int {
	if !((2+3)*4 != 2+3*4) {
		print("RENVO-0275 same_operands_with_different_parentheses_produce_different_values failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
