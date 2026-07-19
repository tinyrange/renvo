package main

func appMain(args []string) int {
	a := 6
	b := 7
	if a*b != 42 {
		print("arithmetic_05 mul\n")
		return 1
	}
	print("PASS\n")
	return 0
}
