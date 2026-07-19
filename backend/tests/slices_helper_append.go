package main

func renvo0566Append(xs []int, n int) []int {
	xs = append(xs, n)
	return xs
}

func appMain(args []string) int {
	var xs []int
	xs = renvo0566Append(xs, 4)
	xs = renvo0566Append(xs, 9)
	if xs[1] != 9 {
		print("RENVO-0566 helper append failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
