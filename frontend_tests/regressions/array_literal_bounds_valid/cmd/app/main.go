package main

type Array [4]int
type Alias = Array

var global = Alias{2: 7, 8, 0: 5, 6}

func main() {
	a := [3]int{2: 9, 0: 7, 8}
	if a[0] != 7 || a[1] != 8 || a[2] != 9 {
		panic("keyed positions")
	}
	b := [...]int{4: 11, 12}
	if len(b) != 6 || b[0] != 0 || b[4] != 11 || b[5] != 12 {
		panic("inferred length")
	}
	if global[0] != 5 || global[1] != 6 || global[2] != 7 || global[3] != 8 {
		panic("named array")
	}
	zero := [0]int{}
	if len(zero) != 0 {
		panic("empty array")
	}
	println("PASS")
}
