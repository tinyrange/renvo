package main

func appMain(args []string) int {
	x := 3
	{
		y := x + 2
		if y != 5 {
			print("RENVO-0315 outer_value_remains_after_shadow_block failed\n")
			return 1
		}
	}
	if !(x == 3) {
		print("RENVO-0315 outer_value_remains_after_shadow_block failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
