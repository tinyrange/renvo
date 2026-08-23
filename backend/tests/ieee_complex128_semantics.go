package main

func appMain(args []string) int {
	leftProduct := float64(2) * 4
	rightProduct := float64(3) * 5
	if leftProduct != 8 || rightProduct != 15 {
		print("FAIL: complex128 scalar products\n")
		return 1
	}
	numerator := leftProduct + rightProduct
	denominator := float64(4)*4 + float64(5)*5
	if numerator != 23 || denominator != 41 {
		print("FAIL: complex128 scalar sums\n")
		return 1
	}
	scalar := numerator / denominator
	if scalar < 0.560975609756 || scalar > 0.560975609757 {
		print("FAIL: complex128 scalar formula\n")
		return 1
	}
	value := complex(float64(2), float64(3)) / complex(float64(4), float64(5))
	if real(value) == 0 {
		print("FAIL: complex128 real division zero\n")
		return 1
	}
	if real(value) < 0 {
		print("FAIL: complex128 real division negative\n")
		return 1
	}
	if real(value) < 0.560975609756 {
		print("FAIL: complex128 real division low\n")
		return 1
	}
	if real(value) > 0.560975609757 {
		print("FAIL: complex128 real division high\n")
		return 1
	}
	if imag(value) < 0.048780487804 || imag(value) > 0.048780487805 {
		print("FAIL: complex128 imaginary division\n")
		return 1
	}

	product := value * complex(float64(4), float64(5))
	if real(product) < 1.999999999999 || real(product) > 2.000000000001 || imag(product) < 2.999999999999 || imag(product) > 3.000000000001 {
		print("FAIL: complex128 multiply\n")
		return 1
	}

	print("PASS\n")
	return 0
}
