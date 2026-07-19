package main

func appMain(args []string) int {
	x := 6.25
	y := x
	if y != 6.25 {
		print("float_literals_10 copy\n")
		return 1
	}
	print("PASS\n")
	return 0
}
