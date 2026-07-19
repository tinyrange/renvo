package main

func appMain(args []string) int {
	x := 2.5 * 4.0
	if x != 10.0 {
		print("float_literals_06 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
