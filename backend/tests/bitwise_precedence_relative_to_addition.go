package main

func appMain(args []string) int {
	if !(1+6&3 == 3) {
		print("RENVO-0211 bitwise_precedence_relative_to_addition failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
