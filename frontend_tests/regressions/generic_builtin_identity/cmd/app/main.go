package main

import "example.com/genericbuiltinidentity/values"

// All of these names hide the universe types in this package.
type bool struct{}
type string struct{}
type int struct{}
type uint struct{}
type uintptr struct{}
type int8 struct{}
type int16 struct{}
type int32 struct{}
type int64 struct{}
type uint8 struct{}
type uint16 struct{}
type uint32 struct{}
type uint64 struct{}
type float32 struct{}
type float64 struct{}
type complex64 struct{}
type complex128 struct{}
type byte struct{}
type rune struct{}
type error struct{}
type any struct{}
type comparable struct{}

func Id[T interface{}](v T) T { return v }

type Box[T interface{}] struct{ Value T }

func (b Box[T]) Get() T                   { return b.Value }
func Wrap[T interface{}](v T) Box[T]      { return Box[T]{v} }
func Closure[T interface{}](v T) func() T { return func() T { return Id(v) } }
func main() {
	values.CheckBool(Id(values.Bool()))
	values.CheckBool(Wrap(values.Bool()).Get())
	values.CheckBool(Closure(values.Bool())())
	values.CheckBool(values.Id(Id(values.Bool())))
	values.CheckString(Id(values.String()))
	values.CheckString(Wrap(values.String()).Get())
	values.CheckString(Closure(values.String())())
	values.CheckString(values.Id(Id(values.String())))
	values.CheckInt(Id(values.Int()))
	values.CheckInt(Wrap(values.Int()).Get())
	values.CheckInt(Closure(values.Int())())
	values.CheckInt(values.Id(Id(values.Int())))
	values.CheckUint(Id(values.Uint()))
	values.CheckUint(Wrap(values.Uint()).Get())
	values.CheckUint(Closure(values.Uint())())
	values.CheckUint(values.Id(Id(values.Uint())))
	values.CheckUintptr(Id(values.Uintptr()))
	values.CheckUintptr(Wrap(values.Uintptr()).Get())
	values.CheckUintptr(Closure(values.Uintptr())())
	values.CheckUintptr(values.Id(Id(values.Uintptr())))
	values.CheckInt8(Id(values.Int8()))
	values.CheckInt8(Wrap(values.Int8()).Get())
	values.CheckInt8(Closure(values.Int8())())
	values.CheckInt8(values.Id(Id(values.Int8())))
	values.CheckInt16(Id(values.Int16()))
	values.CheckInt16(Wrap(values.Int16()).Get())
	values.CheckInt16(Closure(values.Int16())())
	values.CheckInt16(values.Id(Id(values.Int16())))
	values.CheckInt32(Id(values.Int32()))
	values.CheckInt32(Wrap(values.Int32()).Get())
	values.CheckInt32(Closure(values.Int32())())
	values.CheckInt32(values.Id(Id(values.Int32())))
	values.CheckInt64(Id(values.Int64()))
	values.CheckInt64(Wrap(values.Int64()).Get())
	values.CheckInt64(Closure(values.Int64())())
	values.CheckInt64(values.Id(Id(values.Int64())))
	values.CheckUint8(Id(values.Uint8()))
	values.CheckUint8(Wrap(values.Uint8()).Get())
	values.CheckUint8(Closure(values.Uint8())())
	values.CheckUint8(values.Id(Id(values.Uint8())))
	values.CheckUint16(Id(values.Uint16()))
	values.CheckUint16(Wrap(values.Uint16()).Get())
	values.CheckUint16(Closure(values.Uint16())())
	values.CheckUint16(values.Id(Id(values.Uint16())))
	values.CheckUint32(Id(values.Uint32()))
	values.CheckUint32(Wrap(values.Uint32()).Get())
	values.CheckUint32(Closure(values.Uint32())())
	values.CheckUint32(values.Id(Id(values.Uint32())))
	values.CheckUint64(Id(values.Uint64()))
	values.CheckUint64(Wrap(values.Uint64()).Get())
	values.CheckUint64(Closure(values.Uint64())())
	values.CheckUint64(values.Id(Id(values.Uint64())))
	values.CheckFloat32(Id(values.Float32()))
	values.CheckFloat32(Wrap(values.Float32()).Get())
	values.CheckFloat32(Closure(values.Float32())())
	values.CheckFloat32(values.Id(Id(values.Float32())))
	values.CheckFloat64(Id(values.Float64()))
	values.CheckFloat64(Wrap(values.Float64()).Get())
	values.CheckFloat64(Closure(values.Float64())())
	values.CheckFloat64(values.Id(Id(values.Float64())))
	values.CheckComplex64(Id(values.Complex64()))
	values.CheckComplex64(Wrap(values.Complex64()).Get())
	values.CheckComplex64(Closure(values.Complex64())())
	values.CheckComplex64(values.Id(Id(values.Complex64())))
	values.CheckComplex128(Id(values.Complex128()))
	values.CheckComplex128(Wrap(values.Complex128()).Get())
	values.CheckComplex128(Closure(values.Complex128())())
	values.CheckComplex128(values.Id(Id(values.Complex128())))
	values.CheckByte(Id(values.Byte()))
	values.CheckByte(Wrap(values.Byte()).Get())
	values.CheckByte(Closure(values.Byte())())
	values.CheckByte(values.Id(Id(values.Byte())))
	values.CheckRune(Id(values.Rune()))
	values.CheckRune(Wrap(values.Rune()).Get())
	values.CheckRune(Closure(values.Rune())())
	values.CheckRune(values.Id(Id(values.Rune())))
	values.CheckError(Id(values.Error()))
	values.CheckError(Wrap(values.Error()).Get())
	values.CheckError(Closure(values.Error())())
	values.CheckError(values.Id(Id(values.Error())))
	values.CheckAny(Id(values.Any()))
	values.CheckAny(Wrap(values.Any()).Get())
	values.CheckAny(Closure(values.Any())())
	values.CheckAny(values.Id(Id(values.Any())))
	values.CheckInt(Id(42))
	values.CheckBool(Id(true))
	values.CheckString(Id("value"))
	values.CheckFloat64(Id(1.5))
	values.CheckComplex128(Id(1 + 2i))
	{
		type int = string
		values.CheckInt(Id(42))
	}
	print("PASS\n")
}
