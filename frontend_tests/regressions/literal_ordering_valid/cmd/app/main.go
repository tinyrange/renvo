package main

import "fmt"

var equal = (1i - 1i) == 0
var ordered = 1.5 <= 2

func main() {
	var value float64 = 1
	if equal && ordered && 1i != 2i && (2+3i) == (2+3i) && value+0i < 2 && 0 < 0i+value && real(1i) < 2 && 1 < imag(2i) {
		fmt.Print("PASS\n")
		return
	}
	fmt.Print("FAIL\n")
}
