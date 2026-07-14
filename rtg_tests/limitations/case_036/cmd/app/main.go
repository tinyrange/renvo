package main

type A interface { A() }
type B interface { A }
func main() {}
