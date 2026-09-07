package main

type Key int
type Values map[Key]int

var global = Values{1: 7}

func lookup(m Values) int {
	return m[1]
}

func main() {
	m := map[int]int{1: 5}
	n := m
	if n[1] != 5 || lookup(global) != 7 {
		panic("map lookup")
	}
	{
		m := map[string]int{"x": 9}
		if m["x"] != 9 {
			panic("shadow")
		}
	}
	if m[1] != 5 {
		panic("scope end")
	}
	if m := map[string]int{"x": 11}; m["x"] != 11 {
		panic("header")
	}
	if m[1] != 5 {
		panic("header scope end")
	}
	println("PASS")
}
