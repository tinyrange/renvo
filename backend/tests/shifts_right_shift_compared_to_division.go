package main

func appMain(args []string) int {
	if !(40>>3 == 40/8) {
		print("RENVO-0236 right_shift_compared_to_division failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
