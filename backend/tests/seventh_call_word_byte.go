package main

func firstByte(value byte, b, c, d, e, f, g int) byte {
	return value
}

func appMain() int {
	if firstByte(0x55, 2, 3, 4, 5, 6, 7) != 0x55 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
