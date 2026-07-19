package main

func appMain(args []string) int {
	a := 43
	b := 5
	if a/b != 8 {
		print("arithmetic_06 div\n")
		return 1
	}
	print("PASS\n")
	return 0
}
