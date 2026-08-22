package main

func renvo_runtime_CMultiplyShift32(value uint64, multiplier uint64, high *uint64) uint64 {
	const mask = uint64(0xffffffff)
	valueLow := value & mask
	valueHigh := value >> 32
	multiplierLow := multiplier & mask
	multiplierHigh := multiplier >> 32
	lowProduct := valueLow * multiplierLow
	middle1 := valueHigh * multiplierLow
	middle2 := valueLow * multiplierHigh
	highProduct := valueHigh * multiplierHigh
	low := lowProduct + (middle1 << 32)
	carry := uint64(0)
	if low < lowProduct {
		carry = 1
	}
	beforeMiddle2 := low
	low = low + (middle2 << 32)
	if low < beforeMiddle2 {
		carry++
	}
	*high = highProduct + (middle1 >> 32) + (middle2 >> 32) + carry
	return low>>32 | *high<<32
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
