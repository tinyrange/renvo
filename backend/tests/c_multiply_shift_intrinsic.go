package main

func renvo_runtime_CMultiplyShift32(value uint64, multiplier uint64, high *uint64) uint64 {
	return value * multiplier >> 32
}

func appMain(args []string) int {
	high := uint64(0)
	if renvo_runtime_CMultiplyShift32(^uint64(0), 2, &high) != 0x1ffffffff || high != 1 {
		return 1
	}
	high = 99
	if renvo_runtime_CMultiplyShift32(0x100000000, 3, &high) != 3 || high != 0 {
		return 2
	}
	print("PASS\n")
	return 0
}
