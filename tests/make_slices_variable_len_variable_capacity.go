package main

func appMain() int {
	n := 4
	xs := make([]byte, n, n)
	if len(xs) != 4 {
		print("bad len\n")
		return 1
	}
	xs[0] = 'P'
	xs[1] = 'A'
	xs[2] = 'S'
	xs[3] = 'S'
	if xs[0] != 'P' || xs[1] != 'A' || xs[2] != 'S' || xs[3] != 'S' {
		print("bad values\n")
		return 1
	}
	print("PASS\n")
	return 0
}
