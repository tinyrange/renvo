package main

import "example.com/aggregatealiases/callback"

type Arg = struct{ Value int }

const ArrayLength = 1 + 1

type ArrayArg = struct{ Value [ArrayLength]int }
type Reader = interface {
	First() int
	Second() int
}
type ReaderApply func(func(Reader) int, Reader) int
type NamedArg = struct{ Value callback.NamedScalar }
type PrivateArg = struct{ Value callback.PrivateInner }
type OwnPrivateInner = struct{ value int }
type OwnPrivateArg = struct{ Value OwnPrivateInner }
type Value int

func (v Value) First() int  { return int(v) }
func (v Value) Second() int { return 0 }
func Id[T any](v T) T       { return v }

func main() {
	var f func(func(Arg) int) int = Id(callback.Apply)
	if f(func(v Arg) int { return v.Value }) != 42 {
		panic(1)
	}
	var r ReaderApply = Id(callback.ApplyReader)
	if r(func(v Reader) int { return v.First() + v.Second() }, Value(42)) != 42 {
		panic(2)
	}
	var sum func(Arg) int = Id(callback.Sum)
	var boxed any = sum
	if _, ok := boxed.(func(callback.Arg) int); !ok {
		panic(3)
	}
	if _, ok := boxed.(func(NamedArg) int); ok {
		panic(4)
	}
	var private func(func(PrivateArg) int) int = Id(callback.ApplyPrivate)
	if private(func(v PrivateArg) int { return callback.PrivateValue(v.Value) }) != 42 {
		panic(5)
	}
	var privateFunction func(PrivateArg) int = func(v PrivateArg) int { return callback.PrivateValue(v.Value) }
	boxed = privateFunction
	if _, ok := boxed.(func(OwnPrivateArg) int); ok {
		panic(6)
	}
	if _, ok := boxed.(func(callback.PrivateArg) int); !ok {
		panic(7)
	}

	var array func(func(ArrayArg) int) int = Id(callback.ApplyArray)
	if array(func(v ArrayArg) int { return v.Value[0] + v.Value[1] }) != 42 {
		panic(8)
	}
	print("PASS\n")
}
