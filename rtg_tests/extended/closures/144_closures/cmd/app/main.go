package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(8)
	if add(11) == 19 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
