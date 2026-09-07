package main

type S string

func f(v S) {
	_ = cap(v)
}

func main() {}
