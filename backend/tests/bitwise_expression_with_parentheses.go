package main

func appMain(args []string) int {
	if !((0xf0 | (0x0c & 0x03)) == 0xf0) {
		print("RENVO-0210 bitwise_expression_with_parentheses failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
