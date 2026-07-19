package main

func appMain(args []string) int {
	x := 42
	y := -x
	if y != -42 {
		print("arithmetic_11 unary\n")
		return 1
	}
	print("PASS\n")
	return 0
}
