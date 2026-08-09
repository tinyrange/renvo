package main

func appMain(args []string) int {
	value := 35
	digits := [2]int{}
	at := len(digits)
	for value > 0 {
		at--
		digits[at] = value % 10
		value /= 10
	}
	if at != 0 || digits[0] != 3 || digits[1] != 5 || value != 0 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
