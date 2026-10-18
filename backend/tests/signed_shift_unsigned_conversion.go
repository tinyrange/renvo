package main

func arithmeticShift(value uint64, amount int) uint64 {
	return uint64(int64(value) >> amount)
}
func signExtend(value uint64, width int) int64 {
	return int64(value<<(64-width)) >> (64 - width)
}
func arithmeticWidth(value uint64, width, amount int) uint64 {
	return uint64(signExtend(value, width) >> amount)
}

var shiftInputs = []uint64{0xffffffffffffffff, 0x8000000000000000, 0xffffffffffF00000, 0x80000000}

func appMain(args []string) int {
	for _, value := range shiftInputs {
		want := int64(value)
		for amount := 0; amount < 64; amount++ {
			if arithmeticShift(value, amount) != uint64(want) {
				print("FAIL conversion shift\n")
				return 1
			}
			want = want / 2
			if int64(value) < 0 && value&((uint64(1)<<uint(amount+1))-1) != 0 {
				want -= 1
			}
		}
	}
	if arithmeticWidth(0x80000000, 32, 4) != 0xfffffffff8000000 {
		print("FAIL sign extension\n")
		return 1
	}
	print("PASS\n")
	return 0
}
