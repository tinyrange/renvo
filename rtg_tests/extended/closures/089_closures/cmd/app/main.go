package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(4)
	if add(13) == 17 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
