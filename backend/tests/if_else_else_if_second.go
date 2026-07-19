package main

func appMain(args []string) int {
	x := 0
	if false {
		x = 5
	} else if 3 == 3 {
		x = 6
	} else {
		x = 7
	}
	if x != 6 {
		print("RENVO-0356 else-if second failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
