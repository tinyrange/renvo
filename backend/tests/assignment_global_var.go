package main

var renvo0349Global int

func appMain(args []string) int {
	renvo0349Global = 17
	if renvo0349Global != 17 {
		print("RENVO-0349 global assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
