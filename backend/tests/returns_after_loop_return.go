package main

func renvo0541After(xs []int) int {
	total := 0
	for i := 0; i < len(xs); i = i + 1 {
		total = total + xs[i]
	}
	return total
}

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 4)
	xs = append(xs, 5)
	if renvo0541After(xs) != 9 {
		print("RENVO-0541 after loop return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
