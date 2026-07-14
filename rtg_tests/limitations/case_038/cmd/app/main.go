package main

func main() { x := 1; f := func() int { x = x + 1; return x }; _ = f() }
