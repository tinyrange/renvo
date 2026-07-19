package main

func appMain(args []string) int {
	if false {
		print("RENVO-0290 var_inside_else_block failed\n")
		return 1
	} else {
		var x int = 10
		if x != 10 {
			print("RENVO-0290 var_inside_else_block failed\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
