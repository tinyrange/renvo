package main

var renvo0684Value int = 19

func appMain(args []string) int {
	if renvo0684Value != 19 {
		print("RENVO-0684 int global init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
