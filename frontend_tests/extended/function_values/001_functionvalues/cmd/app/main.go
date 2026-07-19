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
	if 1%2 == 1 {
		fn = mul
	}
	corpusOK := apply(fn, 3, 4) == 12
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print("PASS\n")

}
