package main

func appMain(args []string) int {
	if !(2<<4 == 32) {
		print("RENVO-0228 left_shift_by_const failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
