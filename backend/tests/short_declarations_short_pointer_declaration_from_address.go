package main

func appMain(args []string) int {
	x := 9
	p := &x
	if !(*p == 9) {
		print("RENVO-0308 short_pointer_declaration_from_address failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
