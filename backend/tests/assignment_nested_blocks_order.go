package main

func appMain(args []string) int {
	x := 1
	if x == 1 {
		x = 2
		if x == 2 {
			x = 5
		}
	}
	if x != 5 {
		print("RENVO-0350 nested assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
