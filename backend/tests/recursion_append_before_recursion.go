package main

func renvo0520Build(xs []int, n int) []int {
	if n == 0 || len(xs) > 3 {
		return xs
	}
	xs = append(xs, n)
	return renvo0520Build(xs, n-1)
}

func appMain(args []string) int {
	var xs []int
	xs = renvo0520Build(xs, 3)
	if len(xs) != 3 || xs[0] != 3 || xs[2] != 1 {
		print("RENVO-0520 append before recursion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
