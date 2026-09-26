package main

func pair() (int, int) { return 3, 4 }
func add(a, b int) int { return a + b }
func sum(prefix int, values ...int) int {
	for _, value := range values {
		prefix += value
	}
	return prefix
}
func main() {
	values := []int{2, 3}
	if add(pair()) != 7 || sum(1) != 1 || sum(1, 2, 3) != 6 || sum(1, values...) != 6 {
		panic("call arguments")
	}
	println("PASS")
}
