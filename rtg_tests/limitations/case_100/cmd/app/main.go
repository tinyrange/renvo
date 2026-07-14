package main

import _ "embed"
//go:embed data.txt
var data string
func main() { _ = data }
