package main

func appMain(args []string) int {
	x := 6
	x *= 7
	if !(x == 42) {
		print("RENVO-0333 multiplication_compound_assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
