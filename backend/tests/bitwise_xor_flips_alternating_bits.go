package main

func appMain(args []string) int {
	if !(0x55^0x0f == 0x5a) {
		print("RENVO-0203 bitwise_xor_flips_alternating_bits failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
