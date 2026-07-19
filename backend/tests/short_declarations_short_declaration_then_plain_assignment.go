package main

func appMain(args []string) int {
	x := 2
	x = 7
	if !(x == 7) {
		print("RENVO-0318 short_declaration_then_plain_assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
