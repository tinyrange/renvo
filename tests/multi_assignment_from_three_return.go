package main

func rtg1027Triple() (int, int, int) {
	return 3, 1, 4
}

func appMain(args []string) int {
	a := 0
	b := 0
	c := 0
	a, b, c = rtg1027Triple()
	if a*100+b*10+c != 314 {
		print("RTG-1027 assignment from triple failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
