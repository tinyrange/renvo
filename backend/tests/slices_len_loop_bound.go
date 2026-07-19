package main

var renvo0572Limit int = 4

func appMain(args []string) int {
	var xs []int
	for len(xs) < renvo0572Limit {
		xs = append(xs, len(xs))
	}
	sum := 0
	for i := 0; i < len(xs); i = i + 1 {
		sum = sum + xs[i]
	}
	if sum != 6 {
		print("RENVO-0572 len loop bound failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
