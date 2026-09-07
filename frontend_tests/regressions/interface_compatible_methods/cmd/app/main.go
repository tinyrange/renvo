package main

type Argument = int
type I interface{ M(int) int }
type T int

func (t T) M(x Argument) int { return int(t) + x }

type J interface {
	Set(int)
	Get() int
}
type S struct{ value int }

func (s *S) Set(x int) { s.value = x }
func (s *S) Get() int  { return s.value }

var global I = T(3)

func main() {
	var value I = T(4)
	if value.M(2) != 6 || global.M(1) != 4 {
		panic("interface dispatch")
	}
	value = T(7)
	if value.(T) != T(7) {
		panic("assertion")
	}
	s := S{}
	var pointer J = &s
	pointer.Set(9)
	if pointer.Get() != 9 || s.value != 9 {
		panic("pointer method set")
	}
	println("PASS")
}
