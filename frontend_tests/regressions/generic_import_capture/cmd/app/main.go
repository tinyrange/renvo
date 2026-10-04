package main

import "example.com/genericimportcapture/callback"

var initialized int

func init() { initialized = 7 }

type OwnType struct{ N int }

func (v OwnType) init() int { return v.N }

type OwnBox[T any] struct{ Value T }

func (b OwnBox[T]) init() T { return b.Value }

func Param(Unique int) int          { return callback.Unique() }
func Grouped(Unique, Other int) int { return callback.Unique() + Other }
func Result() (Unique int)          { return callback.Unique() }
func Id[T any](v T) T               { return v }

func main() {
	Unique := func() int { return 31 }
	Renvop0_Unique := 37
	Renvop0_Unique_ := 41
	if Id(callback.Unique()) != 17 || Unique() != 31 || Renvop0_Unique != 37 || Renvop0_Unique_ != 41 {
		panic(1)
	}
	f := callback.Unique
	if Id(f)() != 17 {
		panic(2)
	}
	if callback.Id(callback.Unique)() != 17 {
		panic(3)
	}
	if callback.Apply(callback.Unique) != 17 || callback.Apply(Unique) != 31 {
		panic(4)
	}
	if Param(31) != 17 || Grouped(31, 37) != 54 || Result() != 17 {
		panic(5)
	}
	{
		const Unique = 31
		if callback.Unique() != 17 || Id(Unique) != 31 {
			panic(6)
		}
	}
	{
		type Type int
		v := callback.Type{N: 42}
		x := Type(31)
		if Id(v).N != 42 || callback.Id(x) != 31 {
			panic(7)
		}
	}
	Counter := 31
	if callback.Id(callback.Counter) != 17 || Id(Counter) != 31 {
		panic(8)
	}
	nested := func(Unique int) int { return callback.Unique() + Unique }
	if nested(31) != 48 {
		panic(9)
	}
	if callback.Initialized != 5 || initialized != 7 || callback.ReadType() != 42 || (OwnType{43}).init() != 43 {
		panic(10)
	}
	if callback.Read(44) != 44 || (OwnBox[int]{45}).init() != 45 {
		panic(11)
	}
	RenvoGenericInstance_0, RenvoGenericInstance_1, RenvoGenericInstance_2 := 1, 2, 3
	RenvoGenericInstance_0_, RenvoGenericInstance_3, RenvoGenericInstance_4 := 4, 5, 6
	if callback.Id(46) != 46 || RenvoGenericInstance_0+RenvoGenericInstance_1+RenvoGenericInstance_2+RenvoGenericInstance_0_+RenvoGenericInstance_3+RenvoGenericInstance_4 != 21 {
		panic(12)
	}
	snapshot := Id(f)
	reader := func() int { return f() }
	f = callback.Other
	if snapshot() != 17 || Id(f)() != 19 || callback.Id(reader)() != 19 {
		panic(13)
	}
	var inferred = callback.Unique
	if Id(inferred)() != 17 {
		panic(14)
	}
	var one, two = callback.Unique, callback.Other
	if Id(one)() != 17 || Id(two)() != 19 {
		panic(15)
	}
	three, four := callback.Other, callback.Unique
	if callback.Id(three)() != 19 || callback.Id(four)() != 17 {
		panic(16)
	}
	print("PASS\n")
}
