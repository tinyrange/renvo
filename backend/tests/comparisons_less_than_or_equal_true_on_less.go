package main

func appMain(args []string) int {
	if !(3 <= 7) {
		print("RENVO-0181 less_than_or_equal_true_on_less failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
