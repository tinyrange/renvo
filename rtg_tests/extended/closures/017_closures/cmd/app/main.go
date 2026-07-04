package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(0)
	if add(17) == 17 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
