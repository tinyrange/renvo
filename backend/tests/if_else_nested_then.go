package main

func appMain(args []string) int {
	x := 1
	if x == 1 {
		if x < 2 {
			x = 9
		}
	}
	if x != 9 {
		print("RENVO-0358 nested then failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
