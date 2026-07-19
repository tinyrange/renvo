package main

func appMain(args []string) int {
	if !(-13%5 == -3) {
		print("RENVO-0172 negative_dividend_modulo failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
