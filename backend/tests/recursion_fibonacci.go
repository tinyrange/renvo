package main

func renvo0504Check(n int) int {
	return renvo0504Fib(n)
}

func renvo0504Fib(n int) int {
	if n < 2 {
		return n
	}
	return renvo0504Fib(n-1) + renvo0504Fib(n-2)
}

func appMain(args []string) int {
	if renvo0504Check(7) != 13 {
		print("RENVO-0504 fibonacci failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
