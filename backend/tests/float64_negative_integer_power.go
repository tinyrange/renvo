package main

func float64NegativeIntegerPower(base, exponent float64) float64 {
	integerExponent := int64(exponent)
	negative := integerExponent < 0
	if negative {
		integerExponent = -integerExponent
	}
	result := float64(1)
	for integerExponent > 0 {
		if integerExponent&1 != 0 {
			result *= base
		}
		base *= base
		integerExponent >>= 1
	}
	if negative {
		result = 1 / result
	}
	return result
}

func appMain(args []string) int {
	if float64NegativeIntegerPower(float64(2), float64(-2)) != 0.25 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
