package main

type S struct{ X int }

func main() {
	a := S{X: 1}
	b := S{X: 2}
	if a == b || a != (S{X: 1}) || a.X >= b.X {
		panic("struct comparison")
	}
	{
		a := 3
		if a <= 2 {
			panic("shadowing")
		}
	}
	if (struct{ X int }{X: 1}).X >= 2 {
		panic("literal selector")
	}
	if a.X != 1 {
		panic("outer binding")
	}
	println("PASS")
}
