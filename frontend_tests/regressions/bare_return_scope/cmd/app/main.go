package main

import "fmt"

func reused() (x int) {
	x, y := 3, 4
	x += y
	return
}

func ended() (x int) {
	x = 5
	if x := 9; x > 0 {
		_ = x
	}
	return
}

func explicit() (x int) {
	if true {
		x := 11
		return x
	}
	return
}

func main() {
	if reused() == 7 && ended() == 5 && explicit() == 11 {
		fmt.Print("PASS\n")
		return
	}
	fmt.Print("FAIL\n")
}
