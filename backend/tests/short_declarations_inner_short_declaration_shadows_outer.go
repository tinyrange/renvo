package main

func appMain(args []string) int {
	x := 3
	{
		x := 8
		if x != 8 {
			print("RENVO-0314 inner_short_declaration_shadows_outer failed\n")
			return 1
		}
	}
	if !(x == 3) {
		print("RENVO-0314 inner_short_declaration_shadows_outer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
