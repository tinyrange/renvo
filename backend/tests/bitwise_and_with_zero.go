package main

func appMain(args []string) int {
	if !(0xff&0 == 0) {
		print("RENVO-0212 bitwise_and_with_zero failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
