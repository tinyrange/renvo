package main

func renvo0507Sum(xs []int, i int) int {
	if i >= len(xs) {
		return 0
	} else {
		return xs[i] + renvo0507Sum(xs, i+1)
	}
}

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 2)
	xs = append(xs, 4)
	xs = append(xs, 6)
	if renvo0507Sum(xs, 0) != 12 {
		print("RENVO-0507 slice sum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
