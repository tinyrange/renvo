package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(13)
	if add(13) == 26 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
