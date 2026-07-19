package main

func appMain(args []string) int {
	if false {
		print("RENVO-0311 short_declaration_inside_else_block failed\n")
		return 1
	} else {
		x := 5
		if x != 5 {
			print("RENVO-0311 short_declaration_inside_else_block failed\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
