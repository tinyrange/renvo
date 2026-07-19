package main

func runtimeDivisionHelper(value int, divisor int) (int, int) {
	return value / divisor, value % divisor
}

func appMain() int {
	quotient, remainder := runtimeDivisionHelper(65025+127, 255)
	if quotient != 255 || remainder != 127 {
		return 1
	}
	print("PASS\n")
	return 0
}
