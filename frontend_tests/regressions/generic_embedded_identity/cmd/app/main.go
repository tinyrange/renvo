package main

import "example.com/genericembeddedidentity/callback"

type Base = callback.Base
type Arg = struct{ Base }
type PointerArg = struct{ *Base }
type GenericBase[T any] = callback.GenericBase[T]
type GenericArg[T any] = struct{ GenericBase[T] }
type Other = callback.Base
type OtherArg = struct{ Other }
type base = callback.PrivateBase
type OwnPrivateArg = struct{ base }

func Id[T any](v T) T { return v }
func Visit(v Arg) int { return v.Base.Value }
func main() {
	var f func(func(Arg) int) int = Id(callback.Apply)
	if f(func(v Arg) int { return v.Value }) != 42 {
		panic(1)
	}
	if f(func(v Arg) int { return v.Base.Value }) != 42 {
		panic(2)
	}
	if f(Visit) != 42 {
		panic(3)
	}
	var boxed any = Id(callback.Apply)
	apply, ok := boxed.(func(func(Arg) int) int)
	if !ok || apply(Visit) != 42 {
		panic(4)
	}
	arg, ok := Id(callback.Box()).(Arg)
	if !ok || arg.Base.Value != 45 {
		panic(5)
	}
	if _, ok := Id(callback.Box()).(OtherArg); ok {
		panic(6)
	}
	var pointer func(func(PointerArg) int) int = Id(callback.ApplyPointer)
	if pointer(func(v PointerArg) int { return v.Base.Value }) != 43 {
		panic(7)
	}
	ptr, ok := Id(callback.BoxPointer()).(PointerArg)
	if !ok || ptr.Base.Value != 46 {
		panic(8)
	}
	var generic func(func(GenericArg[int]) int) int = Id(callback.ApplyGeneric)
	if generic(func(v GenericArg[int]) int { return v.GenericBase.Value }) != 44 {
		panic(9)
	}
	gen, ok := Id(callback.BoxGeneric()).(GenericArg[int])
	if !ok || gen.GenericBase.Value != 47 {
		panic(10)
	}
	private, ok := Id(callback.BoxPrivate()).(callback.PrivateArg)
	if !ok || private.Value != 48 {
		panic(11)
	}
	if _, ok := Id(callback.BoxPrivate()).(OwnPrivateArg); ok {
		panic(12)
	}
	one := boxed.(func(func(Arg) int) int)
	if one(Visit) != 42 {
		panic(13)
	}
	keyed := Arg{Base: Base{Value: 49}}
	if Id(keyed).Base.Value != 49 {
		panic(14)
	}
	print("PASS\n")
}
