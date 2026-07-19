package main

func appMain(args []string) int {
	x := 10
	if x > 5 {
		x = x*2 - 3
	} else {
		x = x + 99
	}
	if !(x == 17) {
		print("RENVO-0174 arithmetic_reassignment_across_branches failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
