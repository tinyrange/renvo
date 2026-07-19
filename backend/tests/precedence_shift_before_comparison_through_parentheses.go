package main

func appMain(args []string) int {
	if !((1 << 4) == 16) {
		print("RENVO-0254 shift_before_comparison_through_parentheses failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
