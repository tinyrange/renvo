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
	if 4%2 == 1 {
		fn = mul
	}
	corpusOK := false
	if apply(fn, 6, 7) == 13 {
		corpusOK = true
	}
	if corpusOK {
		print("PASS\n")
		return
	}

	print("FAIL\n")
}
