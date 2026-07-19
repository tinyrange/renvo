package main

func pair() (int, int) { return 2, 3 }
func add(a, b int) int { return a + b }

func main() {
	if add(pair()) == 5 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
