package main

func appMain(args []string) int {
	if 6 != 7 {
		// expected
	} else {
		print("RENVO-0189 comparison_used_in_if_condition failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
