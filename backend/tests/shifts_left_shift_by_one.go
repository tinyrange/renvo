package main

func appMain(args []string) int {
	if !(3<<1 == 6) {
		print("RENVO-0227 left_shift_by_one failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
