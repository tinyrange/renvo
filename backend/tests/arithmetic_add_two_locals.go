package main

func appMain(args []string) int {
	a := 12
	b := 30
	if a+b != 42 {
		print("arithmetic_01 add\n")
		return 1
	}
	print("PASS\n")
	return 0
}
