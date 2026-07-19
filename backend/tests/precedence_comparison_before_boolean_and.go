package main

func appMain(args []string) int {
	if !(3 < 4 && 5 < 6) {
		print("RENVO-0255 comparison_before_boolean_and failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
