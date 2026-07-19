package main

func appMain(args []string) int {
	if !(3 < 3 == false) {
		print("RENVO-0180 less_than_false_on_equal_values failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
