package main

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 1)
	xs = append(xs, 2)
	sum := 0
	for i := 0; i < len(xs); i = i + 1 {
		sum = sum + xs[i]
	}
	if sum != 3 {
		print("RENVO-0409 len condition failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
