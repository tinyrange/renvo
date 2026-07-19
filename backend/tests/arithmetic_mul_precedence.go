package main

func appMain(args []string) int {
	x := 2 + 5*8
	if x != 42 {
		print("arithmetic_09 precedence\n")
		return 1
	}
	print("PASS\n")
	return 0
}
