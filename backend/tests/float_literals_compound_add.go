package main

func appMain(args []string) int {
	x := 2.5
	x += 1.5
	if x != 4.0 {
		print("float_literals_22 addassign\n")
		return 1
	}
	print("PASS\n")
	return 0
}
