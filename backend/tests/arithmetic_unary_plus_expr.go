package main

func appMain(args []string) int {
	x := +(20 + 22)
	if x != 42 {
		print("arithmetic_12 plus\n")
		return 1
	}
	print("PASS\n")
	return 0
}
