package main

func appMain(args []string) int {
	if !(false || 7 > 3) {
		print("RENVO-0256 comparison_before_boolean_or failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
