package main

var arithmeticValueCalls int

func arithmeticValueOperand(value int32) int32 {
	arithmeticValueCalls++
	return value
}

func arithmeticValueUnsigned(value uint32) uint32 {
	return value + 513
}

func arithmeticValueMinimum(value int32) int32 {
	return value - -2147483648
}

func appMain() int {
	if arithmeticValueOperand(2147483645)+7 != -2147483644 || arithmeticValueCalls != 1 {
		return 1
	}
	if arithmeticValueUnsigned(4294967293) != 510 || arithmeticValueMinimum(1) != -2147483647 {
		return 2
	}
	print("PASS\n")
	return 0
}
