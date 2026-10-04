package main

import "example.com/genericconstraintvalues/model"

type C interface{ ~int }
type Value[T any] = interface{ Value() T }
type Box[T any] struct{ Item T }

func (b Box[T]) Value() T      { return b.Item }
func Read[T any](v Value[T]) T { return v.Value() }
func Convert[T C](v int) T     { return T(v) }

var comparable = func(v int) int { return v + 1 }

func main() {
	if Read(Value[int](Box[int]{42})) != 42 || Convert[int](41) != 41 {
		panic("interface conversion")
	}
	C, v := func(v int) int { return v + 1 }, 40
	if C(v) != 41 || comparable(41) != 42 {
		panic("local or package shadow")
	}
	if C := func(v int) int { return v + 2 }; C(v) != 42 {
		panic("initializer shadow")
	}
	f := func(C func(int) int) int { return C(41) }
	if f(func(v int) int { return v + 1 }) != 42 {
		panic("callback argument")
	}
	{
		type C = int
		var s struct{ C int }
		s.C = C(42)
		var callback func(C int) int = func(v int) int { return v + 1 }
		if s.C != 42 || callback(41) != 42 {
			panic("local alias")
		}
	}
	if model.Shadow(41) != 42 || model.Read(model.Box(42)) != 42 {
		panic("imported package")
	}
	print("PASS\n")
}
