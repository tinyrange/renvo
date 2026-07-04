package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(5)
	if add(1) == 6 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
