package main

import "example.com/genericpredeclaredembedding/callback"

func Id[T any](v T) T { return v }

type Imported = callback.Arg
type OwnInt = struct{ int }
type Custom struct{ Value int }

func main() {
	{
		a := struct{ bool }{true}
		b := Id(a)
		if b.bool != true {
			panic(1)
		}
		keyed := struct{ bool }{bool: b.bool}
		if Id(keyed).bool != true {
			panic(30)
		}
	}
	{
		a := struct{ string }{"ok"}
		b := Id(a)
		if b.string != "ok" {
			panic(2)
		}
		keyed := struct{ string }{string: b.string}
		if Id(keyed).string != "ok" {
			panic(31)
		}
	}
	{
		a := struct{ int }{42}
		b := Id(a)
		if b.int != 42 {
			panic(3)
		}
		keyed := struct{ int }{int: b.int}
		if Id(keyed).int != 42 {
			panic(32)
		}
	}
	{
		a := struct{ uint }{42}
		b := Id(a)
		if b.uint != 42 {
			panic(4)
		}
		keyed := struct{ uint }{uint: b.uint}
		if Id(keyed).uint != 42 {
			panic(33)
		}
	}
	{
		a := struct{ uintptr }{42}
		b := Id(a)
		if b.uintptr != 42 {
			panic(5)
		}
		keyed := struct{ uintptr }{uintptr: b.uintptr}
		if Id(keyed).uintptr != 42 {
			panic(34)
		}
	}
	{
		a := struct{ byte }{42}
		b := Id(a)
		if b.byte != 42 {
			panic(6)
		}
		keyed := struct{ byte }{byte: b.byte}
		if Id(keyed).byte != 42 {
			panic(35)
		}
	}
	{
		a := struct{ rune }{42}
		b := Id(a)
		if b.rune != 42 {
			panic(7)
		}
		keyed := struct{ rune }{rune: b.rune}
		if Id(keyed).rune != 42 {
			panic(36)
		}
	}
	{
		a := struct{ int8 }{42}
		b := Id(a)
		if b.int8 != 42 {
			panic(8)
		}
		keyed := struct{ int8 }{int8: b.int8}
		if Id(keyed).int8 != 42 {
			panic(37)
		}
	}
	{
		a := struct{ int16 }{42}
		b := Id(a)
		if b.int16 != 42 {
			panic(9)
		}
		keyed := struct{ int16 }{int16: b.int16}
		if Id(keyed).int16 != 42 {
			panic(38)
		}
	}
	{
		a := struct{ int32 }{42}
		b := Id(a)
		if b.int32 != 42 {
			panic(10)
		}
		keyed := struct{ int32 }{int32: b.int32}
		if Id(keyed).int32 != 42 {
			panic(39)
		}
	}
	{
		a := struct{ int64 }{42}
		b := Id(a)
		if b.int64 != 42 {
			panic(11)
		}
		keyed := struct{ int64 }{int64: b.int64}
		if Id(keyed).int64 != 42 {
			panic(40)
		}
	}
	{
		a := struct{ uint8 }{42}
		b := Id(a)
		if b.uint8 != 42 {
			panic(12)
		}
		keyed := struct{ uint8 }{uint8: b.uint8}
		if Id(keyed).uint8 != 42 {
			panic(41)
		}
	}
	{
		a := struct{ uint16 }{42}
		b := Id(a)
		if b.uint16 != 42 {
			panic(13)
		}
		keyed := struct{ uint16 }{uint16: b.uint16}
		if Id(keyed).uint16 != 42 {
			panic(42)
		}
	}
	{
		a := struct{ uint32 }{42}
		b := Id(a)
		if b.uint32 != 42 {
			panic(14)
		}
		keyed := struct{ uint32 }{uint32: b.uint32}
		if Id(keyed).uint32 != 42 {
			panic(43)
		}
	}
	{
		a := struct{ uint64 }{42}
		b := Id(a)
		if b.uint64 != 42 {
			panic(15)
		}
		keyed := struct{ uint64 }{uint64: b.uint64}
		if Id(keyed).uint64 != 42 {
			panic(44)
		}
	}
	{
		a := struct{ float32 }{42}
		b := Id(a)
		if b.float32 != 42 {
			panic(16)
		}
		keyed := struct{ float32 }{float32: b.float32}
		if Id(keyed).float32 != 42 {
			panic(45)
		}
	}
	{
		a := struct{ float64 }{42}
		b := Id(a)
		if b.float64 != 42 {
			panic(17)
		}
		keyed := struct{ float64 }{float64: b.float64}
		if Id(keyed).float64 != 42 {
			panic(46)
		}
	}
	{
		a := struct{ complex64 }{42}
		b := Id(a)
		if b.complex64 != 42 {
			panic(18)
		}
		keyed := struct{ complex64 }{complex64: b.complex64}
		if Id(keyed).complex64 != 42 {
			panic(47)
		}
	}
	{
		a := struct{ complex128 }{42}
		b := Id(a)
		if b.complex128 != 42 {
			panic(19)
		}
		keyed := struct{ complex128 }{complex128: b.complex128}
		if Id(keyed).complex128 != 42 {
			panic(48)
		}
	}
	{
		a := struct{ any }{any(42)}
		b := Id(a)
		if b.any != any(42) {
			panic(20)
		}
		keyed := struct{ any }{any: b.any}
		if Id(keyed).any != any(42) {
			panic(49)
		}
	}
	{
		a := struct{ error }{nil}
		b := Id(a)
		if b.error != nil {
			panic(21)
		}
		keyed := struct{ error }{error: b.error}
		if Id(keyed).error != nil {
			panic(50)
		}
	}
	{
		value := 42
		a := struct{ *int }{&value}
		b := Id(a)
		if *b.int != 42 {
			panic(60)
		}
		*b.int = 43
		if value != 43 {
			panic(61)
		}
	}
	{
		var boxed any = Id(struct{ byte }{42})
		if _, ok := boxed.(struct{ uint8 }); ok {
			panic(62)
		}
		v, ok := boxed.(struct{ byte })
		if !ok || v.byte != 42 {
			panic(63)
		}
	}
	{
		var boxed any = Id(struct{ rune }{42})
		if _, ok := boxed.(struct{ int32 }); ok {
			panic(64)
		}
	}
	{
		type int = Custom
		a := struct{ int }{Custom{44}}
		if Id(a).int.Value != 44 {
			panic(65)
		}
	}
	{
		boxed := Id(callback.Box())
		if _, ok := boxed.(OwnInt); ok {
			panic(66)
		}
		v, ok := boxed.(Imported)
		if !ok || callback.Read(v) != 42 {
			panic(67)
		}
	}
	{
		var f func(func(Imported) int) int = Id(callback.Apply)
		if f(func(v Imported) int { return callback.Read(v) }) != 43 {
			panic(68)
		}
	}
	{
		v := callback.Id(OwnInt{int: 44})
		if v.int != 44 {
			panic(69)
		}
		var boxed any = v
		if _, ok := boxed.(callback.Arg); ok {
			panic(70)
		}
	}
	print("PASS\n")
}
