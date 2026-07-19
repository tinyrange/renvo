package main

func appMain(args []string) int {
	a := 50
	b := 8
	if a-b != 42 {
		print("arithmetic_03 sub\n")
		return 1
	}
	print("PASS\n")
	return 0
}
