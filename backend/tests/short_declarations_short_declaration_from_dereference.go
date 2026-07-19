package main

func appMain(args []string) int {
	x := 19
	p := &x
	y := *p
	if !(y == 19) {
		print("RENVO-0324 short_declaration_from_dereference failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
