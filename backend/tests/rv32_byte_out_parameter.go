package main

func renvoRV32ReadByte(result *byte) bool {
	value := byte(0)
	for bit := 0; bit < 8; bit++ {
		value = value << 1
		if bit == 0 || bit == 7 {
			value = value | 1
		}
	}
	*result = value
	return true
}

func appMain(args []string) int {
	data := [8]byte{0, 0, 0, 0, 0, 0, 0xaa, 0xbb}
	value := byte(0)
	if !renvoRV32ReadByte(&value) {
		print("FAIL: RV32 byte read\n")
		return 1
	}
	data[5] = value
	if data[5] != 0x81 || data[5]>>4 != 8 || data[5]&15 != 1 {
		print("FAIL: RV32 byte out-parameter normalization\n")
		return 1
	}
	print("PASS\n")
	return 0
}
