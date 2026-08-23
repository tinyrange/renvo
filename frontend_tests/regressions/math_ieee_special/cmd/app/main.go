package main

import "math"

func main() {
	if math.Float64bits(0.02) != 0x3f947ae147ae147b {
		print("FAIL: decimal bits\n")
		return
	}
	if math.Float32bits(float32(0.1)) != 0x3dcccccd {
		print("FAIL: binary32 bits\n")
		return
	}
	nan := math.NaN()
	if !math.IsNaN(nan) || !math.IsInf(math.Inf(1), 1) || !math.Signbit(math.Inf(-1)) {
		print("FAIL: special values\n")
		return
	}
	if math.Float64bits(math.Float64frombits(0x7ff8000000000042)) != 0x7ff8000000000042 {
		print("FAIL: payload\n")
		return
	}
	print("PASS\n")
}
