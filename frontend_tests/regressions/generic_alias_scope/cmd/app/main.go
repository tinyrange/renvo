package main

import "example.com/genericaliasscope/model"

func main() {
	if model.Id(42) != 42 || model.Id("text") != "text" || !model.Id(true) {
		panic("parameter aliases")
	}
	a, b := model.Pair(42, 43)
	if a != 42 || b != 43 || model.Result(42) != 42 {
		panic("parameter and result names")
	}
	if model.Named(model.Value(42)) != model.Value(42) {
		panic("named type alias")
	}
	values := model.Id(map[string][]int{"key": {42, 43}})
	if values["key"][0] != 42 || values["key"][1] != 43 {
		panic("container identity")
	}
	box := model.Box[int]{Value: 42}
	if box.Get() != 42 || box.Closure()() != 42 {
		panic("receiver alias")
	}
	if model.Captured(42)(43) != 43 || model.Local(42)() != 42 {
		panic("captured values")
	}
	if model.LocalType(42)() != 42 || model.NamedLocalType(model.Value(42))() != model.Value(42) {
		panic("local type shadows")
	}
	if model.Parameter(42)(43) != 43 || model.ResultClosure(42)() != 42 || model.CapturedResult(42)() != 42 {
		panic("closure parameter and result aliases")
	}
	if model.LocalAlias(42) != 42 || model.AliasClosure(42)() != 42 {
		panic("local aliases")
	}
	if model.SliceClosure(42)()[0] != 42 {
		panic("closure container result")
	}
	if model.Map(42)["key"] != 42 || model.AliasMap(42)["key"] != 42 {
		panic("map type aliases")
	}
	if model.Read(model.Value(42))() != 42 {
		panic("method value cast")
	}
	delta := 2
	f := model.Convert(func(v int) int { return v + delta })
	if f(40) != 42 {
		panic("converted closure")
	}
	delta = 3
	if f(39) != 42 {
		panic("converted capture identity")
	}
	print("PASS\n")
}
