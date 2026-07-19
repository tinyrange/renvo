package main

func appMain(args []string) int {
	x := 1.25 + 2.75
	if x != 4.0 {
		print("float_literals_04 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
