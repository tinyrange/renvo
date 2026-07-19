package main

func appMain(args []string) int {
	if !(!false && true) {
		print("RENVO-0261 unary_not_before_boolean_and failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
