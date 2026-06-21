package main

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 12)
	xs = append(xs, 5)
	n, first := len(xs), xs[0]
	if n*first != 24 {
		print("RTG-1044 len index short failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
