package main

func splitUint64(value uint64) (int, int) {
	return int(value), int(value >> 32)
}

func appMain() int {
	low, high := splitUint64(uint64(0x1122334455667788))
	if uint32(low) != uint32(0x55667788) || uint32(high) != uint32(0x11223344) {
		print("FAIL: uint64 parameter shift\n")
		return 1
	}
	print("PASS\n")
	return 0
}
