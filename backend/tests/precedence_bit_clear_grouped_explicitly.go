package main

func appMain(args []string) int {
	if !((0xff &^ 0x0f) == 240) {
		print("RENVO-0265 bit_clear_grouped_explicitly failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
