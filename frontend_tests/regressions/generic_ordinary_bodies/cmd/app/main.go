package main

import "example.com/genericordinarybodies/model"

func main() {
	if model.Id(42) != 42 || model.Next(model.Box[int]{Value: 41}) != 42 || model.Number(41).Next() != 42 {
		panic("ordinary types and methods")
	}
	if model.Lookup(1) != int('b') || model.Index(2) != 'c' || model.Literal(0)[0] != 'a' {
		panic("untyped string indexing")
	}
	if string(model.Prefix([]byte{'a'}, "bc")) != "abc" || model.Callback("abc")() != "abc!" {
		panic("ordinary builtins and closures")
	}
	print("PASS\n")
}
