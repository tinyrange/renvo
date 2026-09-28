package main

var f = func(x int) int { y := x + 1; return y }
var value = func(x int) int { return x * 2 }(5)

func main() {
	if f(2) != 3 || value != 10 {
		panic("global closures")
	}
	println("PASS")
}
