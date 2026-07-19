package main

func appMain(args []string) int {
	x := 6 // line comment after statement
	if x != 6 {
		print("RENVO-0806 line comment statement failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
