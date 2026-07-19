package main

var renvo0683Value int

func appMain(args []string) int {
	if renvo0683Value != 0 {
		print("RENVO-0683 int zero global failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
