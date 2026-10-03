package main

func negate(v complex128) complex128    { return -v }
func negateSmall(v complex64) complex64 { return -v }
func positive(v complex64) complex64    { return +v }
func appMain(args []string) int {
	v := negate(complex(0.5, -0.25))
	u := positive(negateSmall(complex64(complex(0.5, -0.25))))
	if real(v) != -0.5 || imag(v) != 0.25 || real(u) != -0.5 || imag(u) != 0.25 {
		return 1
	}
	print("PASS\n")
	return 0
}
