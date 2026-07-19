package main

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 4)
	xs[0] += 9
	if xs[0] != 13 {
		print("RENVO-0344 slice compound failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
