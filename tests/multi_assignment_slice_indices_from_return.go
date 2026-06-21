package main

func rtg1029Values() (int, int) {
	return 13, 21
}

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 0)
	xs = append(xs, 0)
	xs[0], xs[1] = rtg1029Values()
	if xs[0]+xs[1] != 34 {
		print("RTG-1029 slice indices from return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
