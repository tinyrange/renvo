package main

import "example.com/aggregatesemantic/callback"

type Arg = struct {
	A int
	B int
}
type Reader = interface {
	Second() int
	First() int
}
type ReaderApply func(func(Reader) int, Reader) int
type Tagged = struct {
	Value int `json:"value"`
}
type Reordered = struct{ B, A int }
type Value int

func (v Value) First() int  { return int(v) }
func (v Value) Second() int { return 0 }
func Id[T any](v T) T       { return v }
func main() {
	var f func(func(Arg) int) int = Id(callback.Apply)
	if f(func(v Arg) int { return v.A + v.B }) != 42 {
		panic(1)
	}
	var r ReaderApply = Id(callback.ApplyReader)
	if r(func(v Reader) int { return v.First() + v.Second() }, Value(42)) != 42 {
		panic(2)
	}
	var tagged func(func(Tagged) int) int = Id(callback.ApplyTagged)
	if tagged(func(v Tagged) int { return v.Value }) != 42 {
		panic(3)
	}
	var sum func(Arg) int = Id(callback.Sum)
	var boxed any = sum
	if _, ok := boxed.(func(callback.Arg) int); !ok {
		panic(4)
	}
	if _, ok := boxed.(func(Reordered) int); ok {
		panic(5)
	}
	print("PASS\n")
}
