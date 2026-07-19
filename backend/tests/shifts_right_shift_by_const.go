package main

func appMain(args []string) int {
	if !(64>>3 == 8) {
		print("RENVO-0232 right_shift_by_const failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
