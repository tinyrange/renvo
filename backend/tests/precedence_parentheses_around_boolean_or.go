package main

func appMain(args []string) int {
	if !((true || false) && false == false) {
		print("RENVO-0258 parentheses_around_boolean_or failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
