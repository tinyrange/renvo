package main

func renvo_runtime_CMultiplyAddShift64(value uint64, multiplier uint64, addend uint64, shift uint32) uint64 {
	return 0
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
