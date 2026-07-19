package main

func appMain(args []string) int {
	x := 4
	y := x + 6
	if !(y == 10) {
		print("RENVO-0316 short_declaration_uses_previous_local failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
