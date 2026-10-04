package main

import "example.com/genericpredeclaredshadow/model"

type Text string

func Local[T ~string](v T) T {
	int := func(v T) T { return v + "!" }
	value := int(v)
	{
		type int = string
		var suffix int = "?"
		value += T(suffix)
	}
	return value
}

func Before[T interface{}]() T {
	int := int(42)
	if int != 42 {
		panic("initializer visibility")
	}
	var zero T
	return zero
}

func main() {
	if model.Transform(Text("x")) != Text("x!x?x#x@x$") || model.Factory[Text]() != Text("NR") {
		panic("package callbacks")
	}
	if model.Identity(Text("x")) != Text("x") || Local(Text("x")) != Text("x!?") || Before[Text]() != Text("") {
		panic("local shadows")
	}
	if model.FoldedFalse[Text]() {
		panic("folded boolean")
	}
	if model.FoldedComplex[Text]() != 1+2i {
		panic("folded complex")
	}
	print("PASS\n")
}
