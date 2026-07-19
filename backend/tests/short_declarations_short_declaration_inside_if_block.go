package main

func appMain(args []string) int {
	if true {
		x := 4
		if x != 4 {
			print("RENVO-0310 short_declaration_inside_if_block failed\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
