package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(3)
	if add(6) == 9 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
