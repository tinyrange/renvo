package main

func appMain(args []string) int {
	s := 2
	if !(32>>s == 8) {
		print("RENVO-0233 right_shift_by_variable failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
