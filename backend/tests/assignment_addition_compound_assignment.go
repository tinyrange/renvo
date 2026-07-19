package main

func appMain(args []string) int {
	x := 3
	x += 4
	if !(x == 7) {
		print("RENVO-0331 addition_compound_assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
