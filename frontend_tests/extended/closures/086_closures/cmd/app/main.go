package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(1)
	if add(10) == 11 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
