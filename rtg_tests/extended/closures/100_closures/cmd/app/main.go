package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(15)
	if add(5) == 20 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
