package main

func appMain(args []string) int {
	if !((0xf0 & 0xcc) == 0xc0) {
		print("RENVO-0205 bitwise_expression_with_hex_constants failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
