package main

func appMain(args []string) int {
	x := 40
	x /= 5
	if !(x == 8) {
		print("RENVO-0334 division_compound_assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
