package main

import "audit/undotted/lib"

func main() {
	if lib.Value() != 42 {
		panic("module import")
	}
	println("PASS")
}
