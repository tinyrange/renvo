package main

var globalArrayBytes [globalArrayByteCount]byte
var globalArrayInts [globalArrayIntCount]int32

const globalArrayIntCount = 2 + 2
const globalArrayByteCount = 8 * 8

func appMain(args []string) int {
	byteIndex := 63
	globalArrayBytes[byteIndex] = 201
	globalArrayBytes[4] = 7
	if globalArrayBytes[byteIndex] != 201 || globalArrayBytes[4] != 7 {
		return 1
	}

	intIndex := 2
	globalArrayInts[0] = -19
	globalArrayInts[intIndex] = 123456
	if globalArrayInts[0] != -19 || globalArrayInts[intIndex] != 123456 {
		return 1
	}
	print("PASS\n")
	return 0
}
