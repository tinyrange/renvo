package main

func appMain(args []string) int {
	x := 9.0 / 2.0
	if x != 4.5 {
		print("float_literals_07 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
