package main

import "example.com/versionlib"

func main() {
	if versionlib.Value() != 42 { panic("dependency") }
	println("PASS")
}
