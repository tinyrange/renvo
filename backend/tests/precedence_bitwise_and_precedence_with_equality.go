package main

func appMain(args []string) int {
	if !(0x0f&0x03 == 3) {
		print("RENVO-0262 bitwise_and_precedence_with_equality failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
