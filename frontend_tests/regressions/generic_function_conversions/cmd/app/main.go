package main

import "example.com/genericfunctionconversions/model"

type byte string

var calls int

func next() model.Callback[int] { calls++; return model.New(2) }
func increment(v int) int       { return v + 1 }

func main() {
	f := model.New(2)
	g := (func(int) int)(f)
	if g(40) != 42 || model.Convert(f)(40) != 42 {
		panic("captured function conversion")
	}
	h := (func(int) int)(f)
	if h(40) != 42 {
		panic("parenthesized function conversion")
	}
	literal := (func(int) int)(func(v int) int { return v + 2 })
	direct := (func(int) int)(increment)
	if literal(40) != 42 || direct(41) != 42 {
		panic("literal and direct conversions")
	}
	once := (func(int) int)(next())
	if calls != 1 || once(40) != 42 {
		panic("conversion evaluation count")
	}
	holder := &f
	indirect := (func(int) int)(*holder)
	mapping := map[int]model.Callback[int]{0: f}
	fromMap := (func(int) int)(mapping[0])
	if indirect(40) != 42 || fromMap(40) != 42 {
		panic("indirect conversions")
	}
	var zero model.Callback[int]
	convertedNil := (func(int) int)(zero)
	plainNil := (func(int) int)(nil)
	nestedNil := (func(int) int)(nil)
	if convertedNil != nil || plainNil != nil || nestedNil != nil || model.Nil[int]() != nil {
		panic("nil conversions")
	}
	if _, ok := model.Read[model.Callback[int]](model.Box(g)); ok {
		panic("conversion identity")
	}
	alias, ok := model.Read[model.Alias[model.Int]](model.Box(g))
	if !ok || alias(40) != 42 {
		panic("generic function alias")
	}
	var aliased func(model.Int) model.Int = func(v model.Int) model.Int { return v + 1 }
	read, ok := model.Read[func(int) int](model.Box(aliased))
	if !ok || read(41) != 42 {
		panic("parameter aliases")
	}
	var bytes func(model.Byte) model.Rune = func(v model.Byte) model.Rune { return model.Rune(v) + 1 }
	readByte, ok := model.Read[func(uint8) int32](model.Box(bytes))
	if !ok || readByte(41) != 42 {
		panic("universe aliases")
	}
	var pointer func([]model.Int) *model.Int = func(v []model.Int) *model.Int { return &v[0] }
	readPointer, ok := model.Read[func([]int) *int](model.Box(pointer))
	if !ok || *readPointer([]int{42}) != 42 {
		panic("nested aliases and pointer result")
	}
	saved := model.Pointer(42)
	if *saved() != 42 || *(func(v int) *int { return &v })(42) != 42 {
		panic("generic pointer closures")
	}
	makePointer := func() func() *int { return func() *int { v := 42; return &v } }
	if *makePointer()() != 42 {
		panic("nested function result")
	}
	shadowed := func(v byte) byte { return v }
	if shadowed(byte("x")) != byte("x") {
		panic("shadowed universe alias")
	}
	if _, ok := model.Read[func(string) string](model.Box(shadowed)); ok {
		panic("defined parameter identity")
	}
	{
		type rune string
		local := func(v rune) rune { return v }
		if local(rune("y")) != rune("y") {
			panic("local type scope")
		}
	}
	print("PASS\n")
}
