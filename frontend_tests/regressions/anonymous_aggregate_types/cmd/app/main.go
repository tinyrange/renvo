package main

type T int

func (t T) M() int { return int(t) }
func method(x any) int {
	switch v := x.(type) {
	case interface{ M() int }:
		return v.M()
	}
	return -1
}
func main() {
	x := struct{ A int }{A: 1}
	y := struct{ A int }{A: 1}
	if x != y || any(x) != any(y) || x.A != 1 {
		panic("anonymous struct identity")
	}
	if method(T(7)) != 7 {
		panic("anonymous interface")
	}
	println("PASS")
}
