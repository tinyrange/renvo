package main

func comparedComplex() complex128     { return complex128(1 + 2i) }
func comparedComplex64() complex64    { return complex64(1 + 2i) }
func comparedRealComplex() complex128 { return complex128(1) }

func appMain() int {
	if comparedComplex() != 1+2i {
		return 1
	}
	if comparedComplex() == 1+3i {
		return 2
	}
	if comparedComplex() == 2+2i {
		return 3
	}
	if 1+2i != comparedComplex() || 1+2i != comparedComplex64() {
		return 4
	}
	if comparedRealComplex() != 1 || comparedRealComplex() != 1.0 {
		return 5
	}
	if comparedComplex() == 1 || comparedComplex() == 1.0 {
		return 6
	}
	print("PASS\n")
	return 0
}
