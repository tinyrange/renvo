package main

func renvo1012Build() ([]int, int) {
	var xs []int
	xs = append(xs, 4)
	xs = append(xs, 6)
	return xs, len(xs)
}

func appMain(args []string) int {
	xs, n := renvo1012Build()
	if n != 2 || xs[0]+xs[1] != 10 {
		print("RENVO-1012 slice and len returns failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
