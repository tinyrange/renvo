package main

func appMain(args []string) int {
	x := 2.0
	x = x * 3.0
	if x != 6.0 {
		print("float_literals_21 reassign\n")
		return 1
	}
	print("PASS\n")
	return 0
}
