package main

import v "example.com/imported_collection_literal/value"

func main() {
	s := []v.Value{v.New(2), v.New(3)}
	a := [1]v.Value{v.New(4)}
	m := map[int]v.Value{1: v.New(5)}
	p := []*v.Value{nil}
	if s[0].Number()+s[1].Number()+a[0].Number()+m[1].Number() != 14 || p[0] != nil {
		panic("collection constructor")
	}
	println("PASS")
}
