package main

func f(x bool) int {
	if x {
		return 1
	}
}
func main() { println(f(false)) }
