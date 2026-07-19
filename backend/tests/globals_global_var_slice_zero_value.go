package main

var renvo0691Values []int

func appMain(args []string) int {
	if len(renvo0691Values) != 0 {
		print("RENVO-0691 slice zero global failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
