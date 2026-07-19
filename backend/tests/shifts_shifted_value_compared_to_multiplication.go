package main

func appMain(args []string) int {
	if !(7<<3 == 7*8) {
		print("RENVO-0235 shifted_value_compared_to_multiplication failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
