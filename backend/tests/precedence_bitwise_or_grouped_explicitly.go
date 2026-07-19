package main

func appMain(args []string) int {
	if !((0x10 | 0x02) == 18) {
		print("RENVO-0263 bitwise_or_grouped_explicitly failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
