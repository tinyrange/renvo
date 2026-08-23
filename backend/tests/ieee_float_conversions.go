package main

func appMain(args []string) int {
	if int(float64(3.75)) != 3 || int(float64(-3.75)) != -3 {
		print("FAIL: float64 to int truncation\n")
		return 1
	}
	wide := float64(1099511627776)
	if int64(wide) != int64(1099511627776) {
		print("FAIL: float64 to int64\n")
		return 1
	}
	if int(float32(3.75)) != 3 {
		print("FAIL: float32 to int\n")
		return 1
	}
	if float64(int64(-17)) != -17 {
		print("FAIL: int64 to float64\n")
		return 1
	}
	if float32(int32(16777217)) != 16777216 {
		print("FAIL: int32 to float32 rounding\n")
		return 1
	}
	u32 := uint32(0xffffffff)
	if float64(u32) != 4294967295 || float32(u32) != 0x1p32 {
		print("FAIL: uint32 to float\n")
		return 1
	}
	u64 := uint64(1) << 63
	if float64(u64) != 0x1p63 {
		print("FAIL: uint64 to float64\n")
		return 1
	}
	if uint64(float64(4294967295)) != uint64(4294967295) {
		print("FAIL: float64 to uint64\n")
		return 1
	}
	print("PASS\n")
	return 0
}
