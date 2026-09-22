package main

type I interface{ M(int) int }
type J interface{ I }
type T int

func (t T) M(x int) int { return int(t) + x }
func main() {
	var x I = T(7)
	f := I.M
	g := J.M
	if f(x, 2) != 9 || I.M(x, 3) != 10 || g(T(4), 5) != 9 {
		panic("interface method expression")
	}
	println("PASS")
}
