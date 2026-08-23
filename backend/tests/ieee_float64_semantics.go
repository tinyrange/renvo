package main

func ieeeFloat64Identity(value float64) float64 { return value }

func ieeeFloat64Unordered(value float64) (bool, bool, bool, bool, bool, bool) {
	return value == value, value != value, value < 1, value <= 1, value > 1, value >= 1
}

func appMain(args []string) int {
	price := ieeeFloat64Identity(0.02)
	if price == 0 {
		print("FAIL: decimal became zero\n")
		return 1
	}
	if price*100 != 2 {
		print("FAIL: decimal arithmetic\n")
		return 1
	}

	one := float64(1)
	zero := float64(0)
	positiveInfinity := one / zero
	negativeInfinity := -one / zero
	nan := zero / zero
	if positiveInfinity <= one || negativeInfinity >= -one || nan == nan {
		print("FAIL: special values\n")
		return 1
	}
	equal, unequal, less, lessEqual, greater, greaterEqual := ieeeFloat64Unordered(nan)
	if equal || !unequal || less || lessEqual || greater || greaterEqual {
		print("FAIL: materialized unordered comparisons\n")
		return 1
	}

	negativeZero := -zero
	if one/negativeZero >= zero {
		print("FAIL: negative zero\n")
		return 1
	}

	smallest := 0x1p-1074
	if smallest == zero || smallest/2 != zero {
		print("FAIL: subnormal rounding\n")
		return 1
	}

	if 0x1.00000000000008p0 != 1 || 0x1.00000000000018p0 == 1 {
		print("FAIL: round to even\n")
		return 1
	}

	maxFinite := 1.7976931348623157e308
	if maxFinite == positiveInfinity || maxFinite*2 != positiveInfinity {
		print("FAIL: overflow boundary\n")
		return 1
	}

	print("PASS\n")
	return 0
}
