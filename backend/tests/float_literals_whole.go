package main

func appMain(args []string) int {
	x := 1.0
	if x != 1.0 {
		print("float_literals_01 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
