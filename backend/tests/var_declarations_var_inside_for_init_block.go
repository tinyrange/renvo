package main

func appMain(args []string) int {
	for i := 0; i < 1; i = i + 1 {
		var x int = i + 6
		if x != 6 {
			print("RENVO-0291 var_inside_for_init_block failed\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
