package main

type T struct{ X int }
func f() int { return 1 }
var x = T{X: f()}
func main() { _ = x }
