package main

func relationMask(a uint, b uint) int {
	result := 0
	if a < b {
		result = result | 1
	}
	if a <= b {
		result = result | 2
	}
	if a > b {
		result = result | 4
	}
	if a >= b {
		result = result | 8
	}
	less, lessEqual, greater, greaterEqual := a < b, a <= b, a > b, a >= b
	if less {
		result = result | 16
	}
	if lessEqual {
		result = result | 32
	}
	if greater {
		result = result | 64
	}
	if greaterEqual {
		result = result | 128
	}
	return result
}

func appMain() int {
	negative := -1
	high := uint(negative)
	if relationMask(high, 1) != 204 {
		return 1
	}
	if relationMask(1, high) != 51 {
		return 2
	}
	if relationMask(high, high) != 170 {
		return 3
	}
	if relationMask(0, 0) != 170 {
		return 4
	}
	if relationMask(high-1, high) != 51 {
		return 5
	}
	if relationMask(high, high-1) != 204 {
		return 6
	}
	if relationMask(0, 1) != 51 {
		return 7
	}
	print("PASS\n")
	return 0
}
