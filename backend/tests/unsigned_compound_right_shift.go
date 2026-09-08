package main

func shift(value uint32, count uint) uint32 {
	value >>= count
	return value
}

func appMain() int {
	if shift(0x80000000, 1) != 0x40000000 || shift(0xffffffff, 31) != 1 || shift(0xffffffff, 32) != 0 {
		print("FAIL unsigned compound right shift\n")
		return 1
	}
	print("PASS\n")
	return 0
}
