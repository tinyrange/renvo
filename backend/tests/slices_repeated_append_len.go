package main

func appMain(args []string) int {
	var xs []int
	for len(xs) < 5 {
		xs = append(xs, len(xs)+1)
	}
	if len(xs) != 5 || xs[4] != 5 {
		print("RENVO-0558 repeated append len failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
