package main

func appMain(args []string) int {
	x := 5 // line comment after statement
	if x != 5 {
		print("RENVO-0805 line comment declaration failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
