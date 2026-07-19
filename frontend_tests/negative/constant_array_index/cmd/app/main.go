package main

type offset int

const base = 1

func main() {
	values := [4]int{}
	_ = values[offset(base+3)]
}
