package main

func uint64CallCompareDivide(dividend uint64, divisor uint32) uint64 {
	return dividend / uint64(divisor)
}

func uint64CallCompare(value uint64, digit uint32, base uint32) bool {
	return value > uint64CallCompareDivide(^uint64(0)-uint64(digit), base)
}

func appMain() int {
	if uint64CallCompare(100, 1, 10) {
		return 1
	}
	print("PASS\n")
	return 0
}
