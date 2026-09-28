package main

func f(values ...int) {}
func main() { f(1, []int{2}...) }
