package main

import "example.com/genericnamespacealiases/callback"

type Base = callback.Base
type Renvop0_Base int
type Renvop0_Base_ string
type base int
type renvop0_base string

var Initialized = 7

func Value() int      { return 19 }
func renvoi0_0() int  { return 29 }
func Id[T any](v T) T { return v }

func main() {
	b, ok := Id(callback.Box()).(Base)
	if !ok || b.Value != 42 {
		panic(1)
	}
	if _, ok := Id(callback.Box()).(Renvop0_Base); ok {
		panic(2)
	}
	x, ok := callback.Id(any(Renvop0_Base(23))).(Renvop0_Base)
	if !ok || x != 23 {
		panic(3)
	}
	if callback.Id(Renvop0_Base_("suffix")) != "suffix" {
		panic(4)
	}
	Renvop0_Value := 31
	Renvop0_Value_ := 37
	if callback.Value() != 17 || Value() != 19 || Renvop0_Value != 31 || Renvop0_Value_ != 37 {
		panic(5)
	}
	if callback.Initialized != 5 || Initialized != 7 || renvoi0_0() != 29 {
		panic(6)
	}
	if _, ok := callback.Id(callback.PrivateBox()).(base); ok {
		panic(7)
	}
	if _, ok := Id(callback.PrivateBox()).(renvop0_base); ok {
		panic(8)
	}
	if Id(renvop0_base("private")) != "private" {
		panic(9)
	}
	print("PASS\n")
}
