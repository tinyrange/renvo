package main

func appMain(args []string) int {
	if !(0x2f&0x0f == 0x0f) {
		print("RENVO-0201 bitwise_and_masks_low_bits failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
