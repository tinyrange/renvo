package main

const shiftConst = 1 << 5

func appMain(args []string) int {
	if !(shiftConst == 32) {
		print("RENVO-0247 shift_in_top_level_const_expression failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
