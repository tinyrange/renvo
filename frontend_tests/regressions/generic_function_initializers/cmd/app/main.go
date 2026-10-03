package main

import "example.com/functioninitializers/callback"

func First() int      { return 17 }
func Second() int     { return 19 }
func Id[T any](v T) T { return v }

var Global = (First)

var GroupFirst, GroupSecond = (First), (Second)

func ReadTypedShadow() int {
	var First func() int = (First)
	return Id(First)()
}
func ReadTypedGroup() int {
	var First, Second func() int = (First), (Second)
	return Id(First)() + Id(Second)()
}

func Factory() func() int { return (First) }
func main() {
	_ = func() int { return 31 }
	f := (First)
	if Id(f)() != 17 {
		panic(1)
	}
	var g = (Second)
	if callback.Id(g)() != 19 {
		panic(2)
	}
	one, two := (First), (Second)
	if Id(one)() != 17 || Id(two)() != 19 {
		panic(3)
	}
	var three, four = (Second), (First)
	if Id(three)() != 19 || callback.Id(four)() != 17 {
		panic(4)
	}
	one, two = two, one
	if Id(one)() != 19 || Id(two)() != 17 {
		panic(5)
	}
	one, newValue := (First), (Second)
	if Id(one)() != 17 || Id(newValue)() != 19 {
		panic(6)
	}
	if Id(Global)() != 17 {
		panic(7)
	}
	type Fn func() int
	defined := Fn((First))
	if Id(defined)() != 17 {
		panic(8)
	}
	imported := callback.Fn((callback.Second))
	if callback.Id(imported).Read() != 19 {
		panic(9)
	}
	empty := Fn((nil))
	if Id(empty) != nil {
		panic(10)
	}
	converted := callback.Fn((func() int { return 23 }))
	if Id(converted).Read() != 23 {
		panic(11)
	}
	made := (Factory)()
	if Id(made)() != 17 {
		panic(12)
	}
	First := (First)
	if callback.Id(First)() != 17 {
		panic(13)
	}
	if Id(GroupFirst)() != 17 || callback.Id(GroupSecond)() != 19 {
		panic(14)
	}
	if ReadTypedShadow() != 17 {
		panic(15)
	}
	if ReadTypedGroup() != 36 {
		panic(16)
	}
	print("PASS\n")
}
