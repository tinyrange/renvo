package main

func rtg1047Pair() (int, int) {
	return 4, 6
}

func rtg1047Use(a int, b int) int {
	return a*b + b
}

func appMain(args []string) int {
	if rtg1047Use(rtg1047Pair()) != 30 {
		print("RTG-1047 direct pair args failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
