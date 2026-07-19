package main

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 1)
	xs = append(xs, 2)
	xs[1] = 7
	if xs[1] != 7 {
		print("RENVO-0343 slice assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
