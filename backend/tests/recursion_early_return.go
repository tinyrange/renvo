package main

func renvo0513Find(n int, stop int) int {
	if n == stop {
		return n
	}
	if n > 9 {
		return -1
	}
	return renvo0513Find(n+1, stop)
}

func appMain(args []string) int {
	if renvo0513Find(1, 6) != 6 {
		print("RENVO-0513 early return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
