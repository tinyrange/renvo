package main

func appMain(args []string) int {
	x := 10
	x -= 4
	if !(x == 6) {
		print("RENVO-0332 subtraction_compound_assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
