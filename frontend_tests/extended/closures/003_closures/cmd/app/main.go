package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(3)
	for corpusAttempt := 0; corpusAttempt < 1; corpusAttempt++ {
		if add(3) == 6 {
			print("PASS\n")
			return
		}
	}

	print("FAIL\n")
}
