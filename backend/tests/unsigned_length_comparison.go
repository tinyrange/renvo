package main

var unsignedLengthCalls int

func unsignedLengthValue(value uint) uint {
	unsignedLengthCalls++
	return value
}

func unsignedLengthLess(value uint, data []byte) bool { return value < uint(len(data)) }
func unsignedLengthASCII(value byte) bool             { return uint(value|32)-'a' < 26 }

func appMain() int {
	data := make([]byte, 3, 7)
	maximum := ^uint(0)
	signBit := maximum/2 + 1
	if unsignedLengthValue(maximum) < uint(len(data)) || !(unsignedLengthValue(signBit) >= uint(cap(data))) {
		panic("unsigned negative bits")
	}
	if unsignedLengthValue(2) >= uint(len(data)) || !(unsignedLengthValue(8) > uint(cap(data))) {
		panic("unsigned positive comparison")
	}
	if unsignedLengthValue(3) > uint(len(data)) || unsignedLengthValue(7) < uint(cap(data)) {
		panic("unsigned equal comparison")
	}
	if unsignedLengthValue(0) > 0 || !(unsignedLengthValue(maximum) > 0) {
		panic("unsigned zero comparison")
	}
	if unsignedLengthValue(signBit) < 26 || unsignedLengthValue(maximum) <= 127 || !(unsignedLengthValue(signBit) > 2147483647) || !(unsignedLengthValue(maximum) >= 1) {
		panic("unsigned constant sign boundary")
	}
	if unsignedLengthValue(25) >= 26 || unsignedLengthValue(26) < 26 || unsignedLengthValue(127) > 127 || unsignedLengthValue(128) <= 127 {
		panic("unsigned constant boundary")
	}
	if unsignedLengthLess(maximum, data) || unsignedLengthLess(signBit, data) || !unsignedLengthLess(2, data) || unsignedLengthLess(3, data) {
		panic("unsigned comparison values")
	}
	if !unsignedLengthASCII('A') || !unsignedLengthASCII('z') || unsignedLengthASCII('1') || unsignedLengthASCII(0) || unsignedLengthASCII(255) {
		panic("unsigned byte comparison values")
	}
	if unsignedLengthCalls != 16 {
		panic("unsigned evaluation count")
	}
	print("PASS\n")
	return 0
}
