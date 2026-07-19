package main

var renvo0522Seed int = 5

func renvo0522Fib(n int) int {
	if n < 2 {
		return n
	}
	return renvo0522Fib(n-1) + renvo0522Fib(n-2)
}

func appMain(args []string) int {
	if renvo0522Fib(renvo0522Seed) != 5 {
		print("RENVO-0522 double recursion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
