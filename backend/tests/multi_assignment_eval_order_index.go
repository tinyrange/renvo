package main

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 4)
	xs = append(xs, 9)
	i := 0
	xs[i], i = 12, 1
	if xs[0] != 12 || xs[1] != 9 || i != 1 {
		print("RENVO-1032 index eval order failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
