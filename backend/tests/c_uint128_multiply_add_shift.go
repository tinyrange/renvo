package main

func renvo_runtime_CMultiplyAddShift64(value uint64, multiplier uint64, addend uint64, shift uint32) uint64 {
	const mask = uint64(0xffffffff)
	valueLow := value & mask
	valueHigh := value >> 32
	multiplierLow := multiplier & mask
	multiplierHigh := multiplier >> 32
	lowProduct := valueLow * multiplierLow
	middle1 := valueHigh * multiplierLow
	middle2 := valueLow * multiplierHigh
	high := valueHigh*multiplierHigh + (middle1 >> 32) + (middle2 >> 32)
	low := lowProduct + (middle1 << 32)
	if low < lowProduct {
		high++
	}
	beforeMiddle2 := low
	low = low + (middle2 << 32)
	if low < beforeMiddle2 {
		high++
	}
	beforeAdd := low
	low = low + addend
	if low < beforeAdd {
		high++
	}
	if shift == 0 {
		return low
	}
	if shift >= 64 {
		return high >> (shift - 64)
	}
	return low>>shift | high<<(64-shift)
}

func appMain(args []string) int {
	if renvo_runtime_CMultiplyAddShift64(10, 20, 3, 4) != 12 ||
		renvo_runtime_CMultiplyAddShift64(^uint64(0), 2, 2, 64) != 2 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
