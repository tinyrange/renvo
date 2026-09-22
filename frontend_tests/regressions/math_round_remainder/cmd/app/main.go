package main

import "math"

func main() {
	if math.Floor(-1.5) != -2 || math.Trunc(-1.5) != -1 || math.Mod(math.MaxFloat64, 3) != 2 {
		panic("rounding")
	}
	if math.Mod(math.SmallestNonzeroFloat64*7, math.SmallestNonzeroFloat64*3) != math.SmallestNonzeroFloat64 {
		panic("subnormal remainder")
	}
	if !math.Signbit(math.Mod(-4, 2)) || !math.IsNaN(math.Mod(1, 0)) || math.MinInt64 != -9223372036854775808 || math.MaxInt != int(^uint(0)>>1) {
		panic("boundary")
	}
	print("PASS\n")
}
