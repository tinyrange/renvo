package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(1)
	corpusOK := add(1) == 2
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print("PASS\n")

}
