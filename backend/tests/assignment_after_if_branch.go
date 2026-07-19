package main

func appMain(args []string) int {
	x := 3
	if x < 5 {
		x = x + 7
	}
	if x != 10 {
		print("RENVO-0336 assignment after if failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
