package main

func appMain(args []string) int {
	x := 7.5
	x -= 2.0
	if x != 5.5 {
		print("float_literals_23 subassign\n")
		return 1
	}
	print("PASS\n")
	return 0
}
