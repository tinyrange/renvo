package values

func Bool() bool { return true }
func CheckBool(v interface{}) {
	x, ok := v.(bool)
	if !ok || x != true {
		panic("bool identity")
	}
}
func String() string { return "value" }
func CheckString(v interface{}) {
	x, ok := v.(string)
	if !ok || x != "value" {
		panic("string identity")
	}
}
func Int() int { return 42 }
func CheckInt(v interface{}) {
	x, ok := v.(int)
	if !ok || x != 42 {
		panic("int identity")
	}
}
func Uint() uint { return 42 }
func CheckUint(v interface{}) {
	x, ok := v.(uint)
	if !ok || x != 42 {
		panic("uint identity")
	}
}
func Uintptr() uintptr { return 42 }
func CheckUintptr(v interface{}) {
	x, ok := v.(uintptr)
	if !ok || x != 42 {
		panic("uintptr identity")
	}
}
func Int8() int8 { return 42 }
func CheckInt8(v interface{}) {
	x, ok := v.(int8)
	if !ok || x != 42 {
		panic("int8 identity")
	}
}
func Int16() int16 { return 42 }
func CheckInt16(v interface{}) {
	x, ok := v.(int16)
	if !ok || x != 42 {
		panic("int16 identity")
	}
}
func Int32() int32 { return 42 }
func CheckInt32(v interface{}) {
	x, ok := v.(int32)
	if !ok || x != 42 {
		panic("int32 identity")
	}
}
func Int64() int64 { return 42 }
func CheckInt64(v interface{}) {
	x, ok := v.(int64)
	if !ok || x != 42 {
		panic("int64 identity")
	}
}
func Uint8() uint8 { return 42 }
func CheckUint8(v interface{}) {
	x, ok := v.(uint8)
	if !ok || x != 42 {
		panic("uint8 identity")
	}
}
func Uint16() uint16 { return 42 }
func CheckUint16(v interface{}) {
	x, ok := v.(uint16)
	if !ok || x != 42 {
		panic("uint16 identity")
	}
}
func Uint32() uint32 { return 42 }
func CheckUint32(v interface{}) {
	x, ok := v.(uint32)
	if !ok || x != 42 {
		panic("uint32 identity")
	}
}
func Uint64() uint64 { return 42 }
func CheckUint64(v interface{}) {
	x, ok := v.(uint64)
	if !ok || x != 42 {
		panic("uint64 identity")
	}
}
func Float32() float32 { return 1.5 }
func CheckFloat32(v interface{}) {
	x, ok := v.(float32)
	if !ok || x != 1.5 {
		panic("float32 identity")
	}
}
func Float64() float64 { return 1.5 }
func CheckFloat64(v interface{}) {
	x, ok := v.(float64)
	if !ok || x != 1.5 {
		panic("float64 identity")
	}
}
func Complex64() complex64 { return 1 + 2i }
func CheckComplex64(v interface{}) {
	x, ok := v.(complex64)
	if !ok || x != 1+2i {
		panic("complex64 identity")
	}
}
func Complex128() complex128 { return 1 + 2i }
func CheckComplex128(v interface{}) {
	x, ok := v.(complex128)
	if !ok || x != 1+2i {
		panic("complex128 identity")
	}
}
func Byte() byte { return 42 }
func CheckByte(v interface{}) {
	x, ok := v.(byte)
	if !ok || x != 42 {
		panic("byte identity")
	}
}
func Rune() rune { return 42 }
func CheckRune(v interface{}) {
	x, ok := v.(rune)
	if !ok || x != 42 {
		panic("rune identity")
	}
}

type fault struct{}

func (fault) Error() string { return "fault" }
func Error() error          { return fault{} }
func CheckError(v interface{}) {
	x, ok := v.(error)
	if !ok || x.Error() != "fault" {
		panic("error identity")
	}
}
func Any() any { return 42 }
func CheckAny(v interface{}) {
	if x, ok := v.(int); !ok || x != 42 {
		panic("any identity")
	}
}
func Id[T any](v T) T { return v }
