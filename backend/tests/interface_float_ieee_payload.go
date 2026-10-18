package main

func floatPayloadEqual(a, b any) bool { return a == b }
func appMain(args []string) int {
	zero := float64(0)
	nan := zero / zero
	f := float32(0)
	nf := f / f
	if floatPayloadEqual(nan, nan) || floatPayloadEqual(nf, nf) || !floatPayloadEqual(zero, -zero) || !floatPayloadEqual(f, -f) {
		print("FAIL interface float equality\n")
		return 1
	}
	a := [1]float64{nan}
	if a == a {
		print("FAIL aggregate NaN\n")
		return 1
	}
	b, c := [1]float32{f}, [1]float32{-f}
	if b != c {
		print("FAIL aggregate zero\n")
		return 1
	}
	print("PASS\n")
	return 0
}
