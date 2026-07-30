package main

func preparedWideSigned(left int64, right int64) int64 {
	value := left / right
	value = value + left%right
	return value
}

func appMain() int {
	left := uint64(0x100000002)
	right := uint64(0x200000103)
	unsigned := left + right
	if unsigned != 0x300000105 {
		return 1
	}
	unsigned = unsigned - right
	if unsigned != left {
		return 4
	}
	unsigned = unsigned * 3
	if unsigned != 0x300000006 {
		return 5
	}
	unsigned = unsigned / 3
	if unsigned != left {
		return 6
	}
	unsigned = unsigned ^ right
	if unsigned != 0x300000101 {
		return 7
	}
	unsigned = unsigned &^ uint64(0xff)
	if unsigned != 0x300000100 {
		return 8
	}
	unsigned = unsigned | 0x40
	if unsigned != 0x300000140 {
		return 9
	}
	unsigned = unsigned << 5
	if unsigned != 0x6000002800 {
		return 10
	}
	unsigned = unsigned >> 3
	if unsigned != 0xc00000500 {
		return 11
	}
	if !(left < right) || !(left <= right) || left > right || left >= right ||
		left == right || !(left != right) {
		return 2
	}
	signed := preparedWideSigned(-0x100000007, 3)
	if signed != -0x55555559 {
		return 3
	}
	print("PASS\n")
	return 0
}
