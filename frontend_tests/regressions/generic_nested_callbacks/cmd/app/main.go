package main

import "example.com/genericnestedcallbacks/callback"
import "example.com/genericnestedcallbacks/semicolon"

func identity[T any](value T) T { return value }

func main() {
	var call func(func(int) int) int = identity(callback.Run)
	if call(func(value int) int { return value }) != 42 {
		panic(1)
	}
	var named callback.Apply = identity(callback.Run)
	if named(func(value int) int { return value }) != 42 {
		panic(2)
	}
	var returning func(func(int) int) func(int) int = identity(callback.Return)
	fn := returning(func(value int) int { return value })
	if fn(42) != 42 {
		panic(3)
	}
	var slice func([]func(int) int) int = identity(callback.Slice)
	values := []func(int) int{func(value int) int { return value }}
	if slice(values) != 42 {
		panic(4)
	}
	base := 40
	defined := identity(callback.Values{func(value int) int { return base + value }})
	if defined[0](2) != 42 {
		panic(5)
	}
	array := [2]callback.Value{1: func(value int) int { return value }}
	if array[1](42) != 42 {
		panic(6)
	}
	entries := map[string]callback.Value{"value": func(value int) int { return value }}
	if entries["value"](42) != 42 {
		panic(7)
	}
	boxed := identity[any](func(value int) int { return value })
	if boxed.(func(int) int)(42) != 42 {
		panic(8)
	}
	imported := identity(semicolon.Apply)
	if imported(42) != 42 {
		panic(9)
	}
	print("PASS\n")
}
