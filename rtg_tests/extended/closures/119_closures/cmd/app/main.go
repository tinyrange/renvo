package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(0)
	if add(5) == 5 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
