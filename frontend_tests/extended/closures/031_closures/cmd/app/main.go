package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(14)
	if add(12) == 26 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
