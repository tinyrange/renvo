package main

func renvo_runtime_CMultiplyDivide64(value uint64, multiplier uint64, divisor uint64) uint64 {
	return value/divisor*multiplier + value%divisor*multiplier/divisor
}

func appMain(args []string) int {
	if renvo_runtime_CMultiplyDivide64(21, 10, 6) != 35 {
		return 1
	}
	if renvo_runtime_CMultiplyDivide64(1<<63, 2, 4) != 1<<62 {
		return 2
	}
	print("PASS\n")
	return 0
}
