package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(12)
	if add(17) == 29 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
