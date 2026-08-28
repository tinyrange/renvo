package main

func pdp11ShiftLeft(value uint16, count uint) uint16 {
	return value << count
}

func pdp11ShiftRight(value uint16, count uint) uint16 {
	return value >> count
}

func pdp11SignedShiftRight(value int16, count uint) int16 {
	return value >> count
}

func appMain(args []string) int {
	_ = args
	if pdp11ShiftLeft(65, 8) != 0x4100 ||
		pdp11ShiftLeft(1, 17) != 0 ||
		pdp11ShiftRight(0x8200, 8) != 0x82 ||
		pdp11ShiftRight(0xffff, 17) != 0 ||
		pdp11SignedShiftRight(-2, 1) != -1 ||
		uint16(0x41ed)&0xf000 != 0x4000 {
		print("FAIL\n")
		return 1
	}
	values := []int{7, 11}
	if values[1] != 11 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
