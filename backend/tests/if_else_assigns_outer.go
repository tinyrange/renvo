package main

func appMain(args []string) int {
	x := 0
	if len(args) >= 0 {
		x = 17
	}
	if x != 17 {
		print("RENVO-0367 outer if assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
