package main

import "fmt"

func main() {
	x :=
		1
	x =
		x + 1
	x +=
		2
	if m := map[int]int{1: 2}; m[1] == 2 {
		x++
	}
	for m := map[int]int{1: 2}; m[1] < 3; m[1]++ {
		x++
	}
	for _, n := range []int{1, 2} {
		x += n
	}
	if x != 9 {
		panic("statement grammar")
	}
	fmt.Println("PASS")
}
