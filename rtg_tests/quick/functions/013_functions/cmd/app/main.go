package main

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func main() {
	if fib(7) == 13 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
