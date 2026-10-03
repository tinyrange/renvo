package main

var lengthComparisonCalls int

func lengthComparisonValue(value int) int {
	lengthComparisonCalls++
	return value
}

func signedLengthLess(value int, data []byte) bool { return value < len(data) }
func signedLengthGreater(value int) bool           { return value > -127 }

func appMain() int {
	data := make([]byte, 3, 7)
	minimum := -int(^uint(0)>>1) - 1
	maximum := int(^uint(0) >> 1)
	if !(lengthComparisonValue(minimum) < len(data)) || lengthComparisonValue(minimum) >= cap(data) {
		panic("negative length comparison")
	}
	if !(lengthComparisonValue(maximum) > len(data)) || lengthComparisonValue(maximum) <= cap(data) {
		panic("positive length comparison")
	}
	if lengthComparisonValue(3) < len(data) || lengthComparisonValue(7) > cap(data) {
		panic("equal length comparison")
	}
	if !(lengthComparisonValue(minimum) < 1) || lengthComparisonValue(minimum) >= 127 || !(lengthComparisonValue(maximum) > -1) || lengthComparisonValue(maximum) <= -127 {
		panic("signed constant opposite signs")
	}
	if lengthComparisonValue(-128) > -127 || lengthComparisonValue(-127) < -127 || lengthComparisonValue(127) > 127 || lengthComparisonValue(128) <= 127 {
		panic("signed constant same signs")
	}
	if !signedLengthLess(minimum, data) || signedLengthLess(maximum, data) || !signedLengthLess(2, data) || signedLengthLess(3, data) {
		panic("signed comparison values")
	}
	if signedLengthGreater(minimum) || !signedLengthGreater(maximum) || signedLengthGreater(-127) || !signedLengthGreater(-126) {
		panic("signed constant comparison values")
	}
	if lengthComparisonCalls != 14 {
		panic("length comparison evaluation")
	}
	print("PASS\n")
	return 0
}
