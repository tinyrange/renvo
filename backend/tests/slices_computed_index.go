package main

const renvo0571Base = 1

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 3)
	xs = append(xs, 6)
	xs = append(xs, 9)
	if xs[renvo0571Base+1] != 9 {
		print("RENVO-0571 computed index failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
