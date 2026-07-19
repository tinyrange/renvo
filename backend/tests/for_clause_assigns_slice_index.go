package main

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 0)
	xs = append(xs, 0)
	for i := 0; i < 2; i = i + 1 {
		xs[i] = i + 6
	}
	if xs[0] != 6 || xs[1] != 7 {
		print("RENVO-0413 slice body failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
