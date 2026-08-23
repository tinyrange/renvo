package main

func renvo_runtime_Float32bits(value float32) uint32     { return 0 }
func renvo_runtime_Float32frombits(value uint32) float32 { return 0 }
func renvo_runtime_Float64bits(value float64) uint64     { return 0 }
func renvo_runtime_Float64frombits(value uint64) float64 { return 0 }

func appMain(args []string) int {
	if renvo_runtime_Float64bits(0.02) != 0x3f947ae147ae147b {
		print("FAIL: float64 decimal bits\n")
		return 1
	}
	if renvo_runtime_Float64bits(0x1p-1074) != 1 {
		print("FAIL: float64 subnormal bits\n")
		return 1
	}
	if renvo_runtime_Float64bits(-0.0) != 0 {
		print("FAIL: constant negative zero\n")
		return 1
	}
	negativeZero := -float64(0)
	if renvo_runtime_Float64bits(negativeZero) != uint64(1)<<63 {
		print("FAIL: float64 negative zero bits\n")
		return 1
	}
	if renvo_runtime_Float32bits(float32(0.1)) != 0x3dcccccd {
		print("FAIL: float32 decimal bits\n")
		return 1
	}
	if renvo_runtime_Float32bits(float32(1e-1)) != 0x3dcccccd {
		print("FAIL: scientific decimal bits\n")
		return 1
	}
	if renvo_runtime_Float32bits(float32(1e-45)) != 1 {
		print("FAIL: subnormal exponent bits\n")
		return 1
	}
	if renvo_runtime_Float32bits(float32(0e0000700)) != 0 {
		print("FAIL: zero exponent bits\n")
		return 1
	}
	if renvo_runtime_Float32bits(renvo_runtime_Float32frombits(1)) != 1 {
		print("FAIL: float32 bit round trip\n")
		return 1
	}
	if renvo_runtime_Float64bits(renvo_runtime_Float64frombits(0x7ff8000000000042)) != 0x7ff8000000000042 {
		print("FAIL: float64 NaN payload round trip\n")
		return 1
	}
	print("PASS\n")
	return 0
}
