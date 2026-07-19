package main

var shiftGlobal int = 1 << 5

func appMain(args []string) int {
	if !(shiftGlobal == 32) {
		print("RENVO-0248 shift_in_global_var_initializer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
