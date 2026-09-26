package main

type I interface{ M() }
type T int

func main() {
	var x I
	_ = x.(T)
}
