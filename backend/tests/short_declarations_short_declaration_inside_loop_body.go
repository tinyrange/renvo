package main

func appMain(args []string) int {
	i := 0
	for i < 1 {
		x := i + 7
		if x != 7 {
			print("RENVO-0313 short_declaration_inside_loop_body failed\n")
			return 1
		}
		i = i + 1
	}
	print("PASS\n")
	return 0
}
