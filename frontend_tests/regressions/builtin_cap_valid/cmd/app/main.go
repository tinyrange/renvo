package main

import "fmt"

func shadowed() int {
	cap := func(s string) int { return len(s) + 10 }
	return cap("abc")
}

func main() {
	values := make([]int, 2, 5)
	channel := make(chan int, 3)
	if cap([2]int{}) == 2 && cap(&[3]int{}) == 3 && cap(values) == 5 && cap(channel) == 3 && shadowed() == 13 {
		fmt.Print("PASS\n")
		return
	}
	fmt.Print("FAIL\n")
}
