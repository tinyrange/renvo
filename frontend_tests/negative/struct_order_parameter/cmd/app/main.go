package main

type S struct{}

func f(a, b S) {
	_ = a <= b
}

func main() {}
