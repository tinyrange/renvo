package main

func renvo0567Mutate(xs []int) {
	if len(xs) == 2 {
		xs[0] = xs[0] + xs[1]
	}
}

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 4)
	xs = append(xs, 8)
	renvo0567Mutate(xs)
	if len(xs) != 2 || xs[0] != 12 {
		print("RENVO-0567 helper mutate failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
