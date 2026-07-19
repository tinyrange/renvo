package main

func renvo1017Add(xs []int, value int) ([]int, int) {
	xs = append(xs, value)
	return xs, xs[len(xs)-1]
}

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 2)
	xs, last := renvo1017Add(xs, 8)
	if len(xs) != 2 || last != 8 || xs[0]+xs[1] != 10 {
		print("RENVO-1017 append result failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
