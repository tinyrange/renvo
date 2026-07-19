package main

func renvo0523Step(n int, keep bool) int {
	if !keep {
		return n
	}
	if n >= 4 {
		return n
	}
	return renvo0523Step(n+1, n+1 < 4)
}

func appMain(args []string) int {
	if renvo0523Step(0, true) != 4 {
		print("RENVO-0523 boolean stop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
