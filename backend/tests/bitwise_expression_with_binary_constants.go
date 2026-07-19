package main

func appMain(args []string) int {
	if !((0b1010 | 0b0101) == 15) {
		print("RENVO-0206 bitwise_expression_with_binary_constants failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
