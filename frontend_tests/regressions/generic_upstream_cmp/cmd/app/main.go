package main

import (
	"fmt"
	cmp "example.com/genericcmp/upstream"
)

type Number int

func main() {
	if !cmp.Less(2, 3) || cmp.Less("z", "a") || cmp.Compare(Number(7), Number(7)) != 0 || cmp.Compare("a", "b") != -1 || cmp.Compare(4, 2) != 1 {
		panic("upstream cmp comparison")
	}
	if cmp.Or(0, 0, 42) != 42 || cmp.Or("", "first", "second") != "first" || cmp.Or[int]() != 0 {
		panic("upstream cmp Or")
	}
	fmt.Println("PASS")
}
