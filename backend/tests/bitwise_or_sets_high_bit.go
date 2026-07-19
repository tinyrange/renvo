package main

func appMain(args []string) int {
	if !(0x20|0x03 == 0x23) {
		print("RENVO-0202 bitwise_or_sets_high_bit failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
