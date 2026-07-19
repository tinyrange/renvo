package main

func appMain(args []string) int {
	if !(0b1000>>2 == 2) {
		print("RENVO-0238 shifted_binary_literal failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
