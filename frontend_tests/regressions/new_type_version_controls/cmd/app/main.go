package main

type Number int
type Record struct{ Value int }

func main() {
	n := new(Number)
	r := new(Record)
	s := new([]int)
	if *n != 0 || r.Value != 0 || *s != nil {
		panic("new type")
	}
	*n = 7
	new := func(n int) int { return n + 1 }
	if new(int(*n)) != 8 {
		panic("shadow")
	}
	println("PASS")
}
