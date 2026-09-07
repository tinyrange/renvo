package main

import "fmt"

type A interface{ Value() int }
type B interface{ Value() int }
type Shared interface {
	A
	B
}
type Explicit interface {
	A
	Value() int
}
type item struct{}

func (item) Value() int { return 7 }

func main() {
	var shared Shared = item{}
	var explicit Explicit = item{}
	if shared.Value() == 7 && explicit.Value() == 7 {
		fmt.Print("PASS\n")
		return
	}
	fmt.Print("FAIL\n")
}
