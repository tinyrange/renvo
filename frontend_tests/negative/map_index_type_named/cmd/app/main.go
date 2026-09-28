package main

type Key int
type M map[Key]int

func main() {
	m := make(M)
	m["x"] = 1
}
