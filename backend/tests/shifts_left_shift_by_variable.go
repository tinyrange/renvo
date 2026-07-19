package main

func appMain(args []string) int {
	s := 3
	if !(5<<s == 40) {
		print("RENVO-0229 left_shift_by_variable failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
