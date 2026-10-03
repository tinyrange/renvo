package main

import "unsafe"
import "example.com/genericunsafelayout/model"

type Inner[T any] struct{ Value T }
type Middle[T any] struct{ Inner[T] }
type Shallow[T any] struct{ Value T }
type Outer[T any] struct {
	Middle[T]
	Shallow[T]
}

type Plain struct{ Value int }

type Own struct{ value int }
type LocalMiddle struct{ Own }
type Cross struct {
	model.Hidden
	model.Blocker
	LocalMiddle
}

func CrossOffset[T any](value Cross) uintptr { return unsafe.Offsetof(value.value) }

var calls int

func Tick[T any](value T) T                { calls++; return value }
func Offset[T any](value Outer[T]) uintptr { return unsafe.Offsetof(value.Value) }
func Size[T any](value T) uintptr          { return unsafe.Sizeof(value) }
func Align[T any](value T) uintptr         { return unsafe.Alignof(value) }

func FixedScalarConstants[T any](value T) int {
	const count = unsafe.Sizeof(uint64(0))
	const mixed = unsafe.Sizeof(float32(0)) + unsafe.Sizeof(byte(0))
	const word = unsafe.Sizeof(int(0))
	var values [count]T
	var bytes [unsafe.Sizeof(complex64(0))]byte
	values[0] = value
	if word != unsafe.Sizeof(int(0)) || mixed != 5 || count != 8 {
		panic("scalar layout constants")
	}
	if _, ok := any(values).([8]T); !ok {
		panic("layout constant array identity")
	}
	return len(values) + len(bytes) + int(mixed)
}

type FixedLayout[T any] struct {
	Values []T
	Last   int
}

type TightLayout struct {
	First byte
	Last  uint64
}

func TightLayoutConstants[T any](value TightLayout) {
	const size = unsafe.Sizeof(value)
	const align = unsafe.Alignof(value)
	const offset = unsafe.Offsetof(value.Last)
	if size != unsafe.Sizeof(value) || align != unsafe.Alignof(value) || offset != unsafe.Offsetof(value.Last) {
		panic("packed scalar layout constants")
	}
}

func FixedLayoutConstants[T any](slice []T, pointer *T, mapping map[int]T, channel chan T) {
	var record FixedLayout[T]
	var boxed any
	const sliceSize = unsafe.Sizeof(slice)
	const pointerSize = unsafe.Sizeof(pointer)
	const mapSize = unsafe.Sizeof(mapping)
	const channelSize = unsafe.Sizeof(channel)
	const interfaceSize = unsafe.Sizeof(boxed)
	const recordSize = unsafe.Sizeof(record)
	const recordAlign = unsafe.Alignof(record)
	const offset = unsafe.Offsetof(record.Last)
	const pointerAlign = unsafe.Alignof(pointer)
	var array [sliceSize]byte
	if sliceSize != unsafe.Sizeof(slice) || pointerSize != unsafe.Sizeof(pointer) || mapSize != unsafe.Sizeof(mapping) || channelSize != unsafe.Sizeof(channel) || interfaceSize != unsafe.Sizeof(boxed) {
		panic("fixed container layout constants")
	}
	if recordSize != unsafe.Sizeof(record) || recordAlign != unsafe.Alignof(record) || offset != unsafe.Offsetof(record.Last) || pointerAlign != unsafe.Alignof(pointer) {
		panic("fixed aggregate layout constants")
	}
	if len(array) != int(unsafe.Sizeof(slice)) {
		panic("container layout array length")
	}
}

func FixedCounts[T any](slice []T, pointer *T) int {
	const first = len([3]uintptr{
		unsafe.Sizeof(Tick(1)),
		unsafe.Alignof(Tick(1)),
		unsafe.Offsetof(Tick(Plain{1}).Value),
	})
	const second = cap([2]uintptr{unsafe.Sizeof(slice), unsafe.Alignof(pointer)})
	var array [first]byte
	return len(array) + second
}

func main() {
	value := Outer[int]{Middle[int]{Inner[int]{7}}, Shallow[int]{17}}
	want := uintptr(unsafe.Pointer(&value.Shallow.Value)) - uintptr(unsafe.Pointer(&value))
	if Offset(value) != want || unsafe.Offsetof((*Outer[int])(nil).Value) != want {
		panic("shallower field offset")
	}
	value.Value = 42
	if value.Shallow.Value != 42 || value.Middle.Value != 7 || value.Value != 42 {
		panic("shallower field selection")
	}
	if Size(value) != unsafe.Sizeof(value) || Align(value) != unsafe.Alignof(value) {
		panic("generic aggregate layout")
	}
	var pointer *Plain
	if unsafe.Offsetof(pointer.Value) != 0 {
		panic("nil base evaluated")
	}
	if FixedCounts([]int{1}, &value.Value) != 5 || FixedCounts([]string{"value"}, (*string)(nil)) != 5 {
		panic("constant enclosing array length")
	}
	if FixedScalarConstants(42) != 21 || FixedScalarConstants("value") != 21 {
		panic("fixed scalar layout constants")
	}
	FixedLayoutConstants([]int{42}, &value.Value, map[int]int{1: 42}, make(chan int, 1))
	FixedLayoutConstants([]string{"value"}, (*string)(nil), map[int]string{}, (chan string)(nil))
	TightLayoutConstants[int](TightLayout{1, 42})
	cross := Cross{model.NewHidden(), model.Blocker(9), LocalMiddle{Own{42}}}
	crossWant := uintptr(unsafe.Pointer(&cross.Own.value)) - uintptr(unsafe.Pointer(&cross))
	if cross.value != 42 || CrossOffset[int](cross) != crossWant {
		panic("foreign private members block field")
	}
	cross.value = 51
	if cross.Own.value != 51 {
		panic("foreign private field assignment")
	}
	if calls != 0 {
		panic("unsafe operand evaluated")
	}
	print("PASS\n")
}
