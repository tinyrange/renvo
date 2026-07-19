package main

func appMain(args []string) int {
	x := 5
	x += 4
	if !(x == 9) {
		print("RENVO-0191 comparison_after_compound_assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
