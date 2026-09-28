package main

type S struct{ X int }
type Outer struct{ Value S }

func main() {
	m := map[int]Outer{}
	m[1].Value.X += 2
}
