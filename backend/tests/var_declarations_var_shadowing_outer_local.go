package main

func appMain(args []string) int {
	var x int = 3
	{
		var x int = 4
		if x != 4 {
			print("RENVO-0292 var_shadowing_outer_local failed\n")
			return 1
		}
	}
	if !(x == 3) {
		print("RENVO-0292 var_shadowing_outer_local failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
