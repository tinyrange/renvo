package main

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 0)
	xs = append(xs, 0)
	i := 0
	for i < len(xs) {
		xs[i] = i + 4
		i = i + 1
	}
	if xs[0] != 4 || xs[1] != 5 {
		print("RENVO-0383 slice write loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
