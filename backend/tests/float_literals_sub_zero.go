package main

func appMain(args []string) int {
	x := 5.5 - 5.5
	if x != 0.0 {
		print("float_literals_05 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
