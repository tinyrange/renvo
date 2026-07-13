package main

func appMain(args []string) int {
	i := 9
	for i := 0; i < 3; i++ {
	}
	if i != 9 {
		print("for_init_short_declaration_shadows_outer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
