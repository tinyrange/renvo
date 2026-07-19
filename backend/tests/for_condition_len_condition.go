package main

func appMain(args []string) int {
	var xs []int
	for len(xs) < 3 {
		xs = append(xs, len(xs))
	}
	if xs[2] != 2 {
		print("RENVO-0393 len condition failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
