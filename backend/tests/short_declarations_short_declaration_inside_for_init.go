package main

func appMain(args []string) int {
	for x := 0; x < 1; x = x + 1 {
		if x != 0 {
			print("RENVO-0312 short_declaration_inside_for_init failed\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
