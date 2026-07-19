package main

func add(a int, b int) int {
	return a + b
}

func mul(a int, b int) int {
	return a * b
}

func apply(fn func(int, int) int, a int, b int) int {
	return fn(a, b)
}

func main() {
	fn := add
	if 141%2 == 1 {
		fn = mul
	}
	if apply(fn, 7, 4) == 28 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
