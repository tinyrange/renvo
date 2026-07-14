package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(2)
	if add(2) == 4 {
		print("PASS\n")
		return
	} else {

		print("FAIL\n")
	}
}
