package main

func renvo0563Fill(xs []int, i int) int {
	if i >= len(xs) {
		return 0
	}
	xs[i] = i + 3
	return xs[i] + renvo0563Fill(xs, i+1)
}

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 0)
	xs = append(xs, 0)
	if renvo0563Fill(xs, 0) != 7 || xs[1] != 4 {
		print("RENVO-0563 read after assign failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
