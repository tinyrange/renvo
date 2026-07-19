package main

func appMain(args []string) int {
	if !(9 > 9 == false) {
		print("RENVO-0184 greater_than_false_on_equal_values failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
