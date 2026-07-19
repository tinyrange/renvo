package main

func appMain(args []string) int {
	x := 8.5
	if x != 8.5 {
		print("float_literals_16 short\n")
		return 1
	}
	print("PASS\n")
	return 0
}
