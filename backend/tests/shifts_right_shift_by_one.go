package main

func appMain(args []string) int {
	if !(8>>1 == 4) {
		print("RENVO-0231 right_shift_by_one failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
