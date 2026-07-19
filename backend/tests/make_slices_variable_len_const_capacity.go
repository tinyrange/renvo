package main

func appMain() int {
	n := 4
	xs := make([]int, n, 16)
	for i := 0; i < len(xs); i++ {
		xs[i] = i + 1
	}
	if len(xs) != 4 {
		print("bad len\n")
		return 1
	}
	if xs[0] != 1 || xs[3] != 4 {
		print("bad values\n")
		return 1
	}
	print("PASS\n")
	return 0
}
