package main

type smallComplex complex64
type largeComplex complex128

func value() complex64 { return complex64(complex(0.5, -0.25)) }
func appMain(args []string) int {
	v := value()
	w := largeComplex(v)
	u := smallComplex(w)
	if real(v) != 0.5 || imag(v) != -0.25 || real(w) != 0.5 || imag(w) != -0.25 || real(u) != 0.5 || imag(u) != -0.25 {
		return 1
	}
	print("PASS\n")
	return 0
}
