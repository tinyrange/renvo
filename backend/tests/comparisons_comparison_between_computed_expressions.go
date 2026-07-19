package main

func appMain(args []string) int {
	if !(2*5 == 7+3) {
		print("RENVO-0187 comparison_between_computed_expressions failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
