package main

func renvo_runtime_COrByte(address *int8, value int8)                {}
func renvo_runtime_CAndByte(address *int8, value int8)               {}
func renvo_runtime_CXorByte(address *int8, value int8)               {}
func renvo_runtime_CXorByteNegative(address *int8, value int8) uint8 { return 0 }
func renvo_runtime_CTestByte(address *uint8, value uint8) uint8      { return 0 }

func renvo_runtime_CBitSet64(address *uint64, bit int64) uint8        { return 0 }
func renvo_runtime_CBitReset64(address *uint64, bit int64) uint8      { return 0 }
func renvo_runtime_CBitComplement64(address *uint64, bit int64) uint8 { return 0 }
func renvo_runtime_CBitTest64(address *uint64, bit int64) uint8       { return 0 }
func renvo_runtime_CBitScanForward32(value uint32, fallback int32) int32 {
	return fallback
}
func renvo_runtime_CBitScanReverse32(value uint32, fallback int32) int32 {
	return fallback
}
func renvo_runtime_CBitScanForward64(value uint64, fallback int64) int64 {
	return fallback
}
func renvo_runtime_CBitScanReverse64(value uint64, fallback int64) int64 {
	return fallback
}

func appMain(args []string) int {
	value := int8(2)
	renvo_runtime_COrByte(&value, 4)
	renvo_runtime_CAndByte(&value, 6)
	renvo_runtime_CXorByte(&value, 3)
	if value != 5 {
		return 1
	}
	value = 0
	if renvo_runtime_CXorByteNegative(&value, -128) != 1 {
		return 6
	}
	renvo_runtime_CXorByte(&value, -128)
	if value != 0 {
		return 7
	}
	byteValue := uint8(8)
	if renvo_runtime_CTestByte(&byteValue, 8) != 1 || renvo_runtime_CTestByte(&byteValue, 4) != 0 {
		return 8
	}

	word := uint64(2)
	if renvo_runtime_CBitSet64(&word, 1) != 1 || word != 2 {
		return 2
	}
	if renvo_runtime_CBitReset64(&word, 1) != 1 || word != 0 {
		return 3
	}
	if renvo_runtime_CBitComplement64(&word, 4) != 0 || word != 16 {
		return 4
	}
	if renvo_runtime_CBitTest64(&word, 4) != 1 || word != 16 {
		return 5
	}
	if renvo_runtime_CBitScanForward32(0x40, -1) != 6 || renvo_runtime_CBitScanReverse32(0x40, -1) != 6 {
		return 9
	}
	if renvo_runtime_CBitScanForward64(1<<41, -1) != 41 || renvo_runtime_CBitScanReverse64(1<<41, -1) != 41 {
		return 10
	}
	zero32 := renvo_runtime_CBitScanForward32(0, -7)
	if zero32 != -7 {
		return 11
	}
	if renvo_runtime_CBitScanReverse64(0, -9) != -9 {
		return 12
	}
	print("PASS\n")
	return 0
}
