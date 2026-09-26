package main

type S struct{ X int }
type Outer struct{ Pointer *S }
type Pointers map[int]*S

func main() {
	p := &S{X: 1}
	m := map[int]*S{1: p}
	m[1].X = 2
	m[1].X++
	if p.X != 3 {
		panic("pointer map")
	}
	nested := map[int]Outer{1: {Pointer: p}}
	nested[1].Pointer.X += 4
	if p.X != 7 {
		panic("nested pointer")
	}
	named := Pointers{1: p}
	named[1].X = 9
	values := map[int]S{1: {X: 1}}
	copy := values[1]
	copy.X = 5
	if values[1].X != 1 {
		panic("map value copy")
	}
	values[1] = copy
	if values[1].X != 5 || p.X != 9 {
		panic("map value replacement")
	}
	println("PASS")
}
