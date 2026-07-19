package main

func appMain(args []string) int {
	x := 0
	if 1 == 1 {
		x = 5
	} else if true {
		x = 6
	} else {
		x = 7
	}
	if x != 5 {
		print("RENVO-0355 else-if first failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
