package main

func appMain(args []string) int {
	if !(-13/5 == -2) {
		print("RENVO-0171 negative_dividend_division failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
