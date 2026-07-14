package main

func apply(f func(int) int) int { return f(1) }
func inc(x int) int { return x+1 }
func main() { f := inc; _ = apply(f) }
