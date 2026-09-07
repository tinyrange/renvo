package main

type Pointer *int

func main() {
	a := new(int)
	b := new(int)
	*a = 3
	*b = 7
	if a == b || a == nil || b == nil || !(*a < *b) {
		panic("pointer equality or dereference")
	}
	alias := a
	if alias != a {
		panic("alias equality")
	}
	pp := &a
	if !(**pp < *b) {
		panic("nested dereference")
	}
	{
		a := 2
		if !(a < 4) {
			panic("shadow")
		}
	}
	if a == nil || *a != 3 {
		panic("scope")
	}
	var named Pointer = a
	if named != a || !(*named < *b) {
		panic("named pointer")
	}
	println("PASS")
}
