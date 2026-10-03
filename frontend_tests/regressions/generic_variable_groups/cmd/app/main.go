package main

import "example.com/variablegroups/callback"

func First() int          { return 17 }
func Second() int         { return 19 }
func Id[T any](v T) T     { return v }
func Pair() (int, string) { return 23, "yes" }

var (
	GlobalFirst          = (First)
	GlobalSecond         = (callback.Second)
	GlobalOne, GlobalTwo = (callback.First), (Second)
)

var (
	Counter = 0
	Before  = Next()
	After   = Next()
)

func Next() int { Counter++; return Counter }
func InferredShadow() int {
	var (
		First = (First)
	)
	return callback.Id(First)()
}
func TypedShadow() int {
	var (
		First func() int = (First)
	)
	return callback.Id(First)()
}
func Grouped[T any](v T) T {
	var (
		first  = v
		second = Id(first)
	)
	return second
}

func main() {
	_ = func() int { return 31 }
	var (
		f = (First)
		g = (callback.Second)
	)
	if Id(f)() != 17 || callback.Id(g)() != 19 {
		panic(1)
	}
	if TypedShadow() != 17 {
		panic(2)
	}
	if InferredShadow() != 17 {
		panic(3)
	}
	var (
		one, two = (callback.First), (Second)
	)
	if Id(one)() != 17 || callback.Id(two)() != 19 {
		panic(4)
	}
	if Id(GlobalFirst)() != 17 || Id(GlobalSecond)() != 19 {
		panic(5)
	}
	if Id(GlobalOne)() != 17 || callback.Id(GlobalTwo)() != 19 {
		panic(6)
	}
	var (
		read = func() int {
			var (
				n = 17
				m = 19
			)
			return Id(n) + m
		}
		value = 23
	)
	if callback.Id(read)() != 36 || Id(value) != 23 {
		panic(7)
	}
	var (
		n, text = Pair()
	)
	if Id(n) != 23 || callback.Id(text) != "yes" {
		panic(8)
	}
	if Grouped(29) != 29 || Grouped("value") != "value" {
		panic(9)
	}
	if Before != 1 || After != 2 || Counter != 2 {
		panic(10)
	}
	print("PASS\n")
}
