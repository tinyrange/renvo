package main

var floatLit19Global float64 = 11.25

func appMain(args []string) int {
	if floatLit19Global != 11.25 {
		print("float_literals_19 global\n")
		return 1
	}
	print("PASS\n")
	return 0
}
