package main

func shiftedPointerValue(value *uint32) uint32 {
	return *value >> 24
}

func appMain(args []string) int {
	value := uint32(0xf5000000)
	if shiftedPointerValue(&value) != 0xf5 {
		return 1
	}
	print("PASS\n")
	return 0
}
