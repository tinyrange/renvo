package main

func appMain(args []string) int {
	a := 0
	a, a = 1, 2
	if a != 2 {
		print("RTG-1033 same target assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
