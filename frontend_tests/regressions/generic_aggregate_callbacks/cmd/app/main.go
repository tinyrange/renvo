package main

import (
	"example.com/genericaggregatecallbacks/callback"
	"example.com/genericaggregatecallbacks/other"
)

type Arg = struct{ Value int }
type Apply func(func(Arg) Arg) int
type PrivateApply func(func(callback.Private) callback.Private) int
type ReaderApply func(func(interface{ Value() int }) int, interface{ Value() int }) int
type Value int

func (v Value) Value() int { return int(v) }
func Id[T any](v T) T      { return v }

func main() {
	var apply Apply = Id(callback.Apply)
	if apply(func(v struct{ Value int }) struct{ Value int } { return v }) != 42 {
		panic(1)
	}
	var plain func(func(struct{ Value int }) struct{ Value int }) int = Id(callback.Apply)
	if plain(func(v Arg) Arg { return v }) != 42 {
		panic(2)
	}
	var private PrivateApply = Id(callback.ApplyPrivate)
	if private(func(v callback.Private) callback.Private { return v }) != 42 {
		panic(3)
	}
	var reader ReaderApply = Id(callback.ApplyReader)
	if reader(func(v callback.Reader) int { return v.Value() }, Value(42)) != 42 {
		panic(4)
	}
	var left func(callback.Private) int = Id(callback.PrivateValue)
	var right func(other.Private) int = Id(other.PrivateValue)
	var boxed any = left
	if _, ok := boxed.(func(other.Private) int); ok {
		panic(5)
	}
	if _, ok := boxed.(func(callback.Private) int); !ok {
		panic(6)
	}
	boxed = right
	if _, ok := boxed.(func(callback.Private) int); ok {
		panic(7)
	}
	print("PASS\n")
}
