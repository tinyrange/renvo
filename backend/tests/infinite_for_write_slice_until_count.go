package main

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 0)
	xs = append(xs, 0)
	i := 0
	for {
		if i == len(xs) {
			break
		}
		xs[i] = i + 8
		i = i + 1
	}
	if xs[1] != 9 {
		print("RENVO-0438 slice write infinite failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
