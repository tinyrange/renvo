package main

func nativeIntShift(value int, count uint) int {
	return value >> count
}

func appMain() int {
	if ^uint(0)>>32 != 0 {
		print("PASS\n")
		return 0
	}
	if nativeIntShift(0x11223344, 32) != 0 || nativeIntShift(-7, 32) != -1 ||
		0x11223344<<uint(32) != 0 {
		print("FAIL: native-width shift\n")
		return 1
	}
	print("PASS\n")
	return 0
}
