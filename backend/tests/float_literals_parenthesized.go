package main

func appMain(args []string) int {
	x := (1.5 + 2.5) * (3.0 - 1.0)
	if x != 8.0 {
		print("float_literals_12 paren\n")
		return 1
	}
	print("PASS\n")
	return 0
}
