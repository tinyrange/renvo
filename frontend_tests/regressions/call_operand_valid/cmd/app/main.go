package main

import "fmt"

func pair() (int, int)      { return 2, 3 }
func sum(a, b int) int      { return a + b }
func forwarded() (int, int) { return pair() }
func one() int              { return 4 }

func main() {
	a, b := forwarded()
	if a == 2 && b == 3 && sum(pair())+1 == 6 && (one())+1 == 5 {
		fmt.Print("PASS\n")
		return
	}
	fmt.Print("FAIL\n")
}
