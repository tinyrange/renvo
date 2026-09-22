package main

type M map[int]int

func main() {
	var empty M
	delete(empty, 1)
	if len(empty) != 0 {
		panic("nil map")
	}
	println("PASS")
}
