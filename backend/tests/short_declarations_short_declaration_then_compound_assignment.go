package main

func appMain(args []string) int {
	x := 2
	x += 5
	if !(x == 7) {
		print("RENVO-0319 short_declaration_then_compound_assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
