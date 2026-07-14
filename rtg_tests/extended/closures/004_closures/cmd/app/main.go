package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(4)
	corpusOK := false
	if add(4) == 8 {
		corpusOK = true
	}
	if corpusOK {
		print("PASS\n")
		return
	}

	print("FAIL\n")
}
