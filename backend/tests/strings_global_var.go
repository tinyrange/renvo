package main

var strings15Global string = "global"

func appMain(args []string) int {
	if strings15Global != "global" {
		print("strings_15 global\n")
		return 1
	}
	print("PASS\n")
	return 0
}
