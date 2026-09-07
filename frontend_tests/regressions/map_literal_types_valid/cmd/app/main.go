package main

import "fmt"

type Key int
type Values map[Key]string
type Alias = Values

var values = Alias{1: "one", 2: "two"}

func main() {
	bytes := map[int]byte{1: "x"[0]}
	if values[1] == "one" && values[2] == "two" && bytes[1] == 120 {
		fmt.Print("PASS\n")
		return
	}
	fmt.Print("FAIL\n")
}
