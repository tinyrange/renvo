package main

import "fmt"

func integers() (first int, second int) {
	first, second = 4, 9
	return second, first
}

func strings() (first string, second string) {
	first, second = "first", "second"
	return second, first
}

func arrays() (first [2]int, second [2]int) {
	first = [2]int{1, 2}
	second = [2]int{3, 4}
	return second, first
}

func calls() (first int, second int) {
	first, second = 4, 9
	defer func() { first++ }()
	return func() int { first = 8; return second }(), func() int { return first }()
}

func main() {
	i, j := integers()
	s, t := strings()
	a, b := arrays()
	x, y := calls()
	if i == 9 && j == 4 && s == "second" && t == "first" && a == [2]int{3, 4} && b == [2]int{1, 2} && x == 10 && y == 8 {
		fmt.Print("PASS\n")
		return
	}
	fmt.Print("FAIL\n")
}
