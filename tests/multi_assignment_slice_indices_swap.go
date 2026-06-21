package main

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 5)
	xs = append(xs, 12)
	xs[0], xs[1] = xs[1], xs[0]
	if xs[0] != 12 || xs[1] != 5 {
		print("RTG-1024 slice index swap failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
