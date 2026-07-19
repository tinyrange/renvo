package main

func appMain(args []string) int {
	x := (2 + 5) * 6
	if x != 42 {
		print("arithmetic_10 paren\n")
		return 1
	}
	print("PASS\n")
	return 0
}
