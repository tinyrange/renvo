package main

func renvo_runtime_Float64bits(value float64) uint64 { return 0 }

func appMain(args []string) int {
	if renvo_runtime_Float64bits(0.560975609757) != 0x3fe1f3831f3851b4 {
		print("FAIL: long decimal literal\n")
		return 1
	}
	print("PASS\n")
	return 0
}
