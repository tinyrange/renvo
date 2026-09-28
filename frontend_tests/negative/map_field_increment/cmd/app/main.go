package main

type S struct{ X int }

func main() {
	m := map[int]S{1: {}}
	m[1].X++
}
