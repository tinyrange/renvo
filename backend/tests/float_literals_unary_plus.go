package main

func appMain(args []string) int {
	x := 4.5
	y := +x
	if y != 4.5 {
		print("float_literals_14 plus\n")
		return 1
	}
	print("PASS\n")
	return 0
}
