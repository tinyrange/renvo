package main

func appMain(args []string) int {
	x := 6.25
	y := x + 1.0
	if y == x {
		print("float_literals_11 neq\n")
		return 1
	}
	print("PASS\n")
	return 0
}
