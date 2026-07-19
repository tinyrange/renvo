package main

func appMain(args []string) int {
	if !(0x10<<2 == 0x40) {
		print("RENVO-0237 shifted_hex_literal failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
