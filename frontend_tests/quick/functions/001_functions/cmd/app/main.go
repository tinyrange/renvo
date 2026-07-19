package main

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func main() {
	corpusOK := fib(4) == 3
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print("PASS\n")

}
