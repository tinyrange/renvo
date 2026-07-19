package main

func appMain(args []string) int {
	if true {
		var x int = 9
		if x != 9 {
			print("RENVO-0289 var_inside_if_block failed\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
