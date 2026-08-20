package main

func appMain() int {
	var word uint32 = 0x12345678
	var small byte = 0x5a
	var wide uint64 = 0x0123456789abcdef
	var signed int = 5
	choice := (word & 0xff00ff00) ^ (^word & 0x00ff00ff)
	if ^word == uint32(0xedcba987) && ^small == byte(0xa5) &&
		^wide == uint64(0xfedcba9876543210) && ^signed == -6 &&
		choice == uint32(0x12cb5687) {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
