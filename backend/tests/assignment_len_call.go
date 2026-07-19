package main

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 3)
	xs = append(xs, 4)
	n := 0
	n = len(xs)
	if n != 2 {
		print("RENVO-0347 len assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
