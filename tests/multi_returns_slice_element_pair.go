package main

func rtg1019Ends(xs []int) (int, int) {
	return xs[0], xs[len(xs)-1]
}

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 6)
	xs = append(xs, 9)
	first, last := rtg1019Ends(xs)
	if first != 6 || last != 9 {
		print("RTG-1019 slice element pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
