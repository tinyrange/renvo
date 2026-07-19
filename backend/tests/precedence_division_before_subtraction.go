package main

func appMain(args []string) int {
	if !(20-8/2 == 16) {
		print("RENVO-0252 division_before_subtraction failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
