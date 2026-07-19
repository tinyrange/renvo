package main

func appMain(args []string) int {
	{
		var x int = 3
		if x != 3 {
			print("RENVO-0288 var_inside_appmain_block failed\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
