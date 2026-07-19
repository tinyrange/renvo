package main

func renvo1047Pair() (int, int) {
	return 4, 6
}

func renvo1047Use(a int, b int) int {
	return a*b + b
}

func appMain(args []string) int {
	if renvo1047Use(renvo1047Pair()) != 30 {
		print("RENVO-1047 direct pair args failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
