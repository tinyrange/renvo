package main

func appMain(args []string) int {
	x := 1
	{
		y := 2
		{
			z := x + y
			if z != 3 {
				print("RENVO-0317 short_declaration_in_nested_block failed\n")
				return 1
			}
		}
	}
	print("PASS\n")
	return 0
}
