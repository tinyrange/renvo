package main

func appMain(args []string) int {
	left := complex(float64(2), float64(3))
	right := complex(float64(4), float64(5))
	product := left * right
	if real(product) != -7 || imag(product) != 22 {
		print("FAIL: complex multiplication\n")
		return 1
	}
	quotient := complex(float64(8), float64(12)) / complex(float64(4), float64(0))
	if real(quotient) != 2 {
		print("FAIL: complex division real component\n")
		return 1
	}
	if imag(quotient) != 3 {
		print("FAIL: complex division imaginary component\n")
		return 1
	}
	print("PASS\n")
	return 0
}
