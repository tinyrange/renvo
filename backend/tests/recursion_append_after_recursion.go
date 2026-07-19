package main

const renvo0521Start = 3

func renvo0521Build(xs []int, n int) []int {
	if n == 0 {
		return xs
	}
	xs = renvo0521Build(xs, n-1)
	xs = append(xs, n)
	return xs
}

func appMain(args []string) int {
	var xs []int
	xs = renvo0521Build(xs, renvo0521Start)
	if len(xs) != 3 || xs[0] != 1 || xs[2] != 3 {
		print("RENVO-0521 append after recursion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
