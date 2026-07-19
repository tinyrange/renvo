package main

func appMain(args []string) int {
	x := 1
	if x == 2 {
		x = 3
	} else {
		if x == 1 {
			x = 4
		}
	}
	if x != 4 {
		print("RENVO-0359 nested else failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
