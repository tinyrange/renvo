package main

func appMain(args []string) int {
	x := 0xf0
	x = x & 0xcc
	x = x | 3
	if !(x == 0xc3) {
		print("RENVO-0221 compound_style_through_explicit_assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
