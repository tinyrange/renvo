package main

type S struct{ X int }
type Alias = S
type Embedded struct{ S }

func (s S) Value() int { return s.X }
func (s *S) Increment() { s.X++ }

func main() {
	s := S{3}
	s.Increment()
	if s.X != 4 || s.Value() != 4 || (S{5}).Value() != 5 {
		panic("members")
	}
	a := Alias{6}
	e := Embedded{S{7}}
	if a.X != 6 || e.X != 7 || e.Value() != 7 {
		panic("alias and promotion")
	}
	{
		s := struct{ Y int }{8}
		if s.Y != 8 {
			panic("shadowing")
		}
	}
	if s.X != 4 {
		panic("outer")
	}
	println("PASS")
}
