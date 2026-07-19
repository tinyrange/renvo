package main

func appMain(args []string) int {
	if !((0x55 ^ 0x0f) == 90) {
		print("RENVO-0264 bitwise_xor_grouped_explicitly failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
