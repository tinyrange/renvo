package main

import "unsafe"

type Empty struct{}
type Receiver[T any] [0]T

func (r Receiver[T]) Value(value T) T { called++; return value }

type Record[T any] struct {
	Before int
	Zero   [0]T
	After  int
}

func Identity[T any]() any { return [unsafe.Sizeof(struct{}{})]T{} }

func Zero[T any]() [0]T { return [0]T{} }

func Choose[T any](empty [0]T, value T) T { return value }

func Variadic[T any](values ...[0]T) int { return len(values) }

var called int

func Tick[T any]() [0]T { called++; return Zero[T]() }

func MakePanics[T any](length, capacity int) (caught bool) {
	defer func() { caught = recover() != nil }()
	values := make([]T, length, capacity)
	_ = values
	return
}

func Check[T any](value T) {
	if _, ok := Identity[T]().([0]T); !ok {
		panic("zero-size array identity")
	}
	if unsafe.Sizeof(Zero[T]()) != 0 {
		panic("zero-size instantiated array")
	}
	var record Record[T]
	record.Before, record.After = 17, 51
	record.Zero = Tick[T]()
	if unsafe.Offsetof(record.Zero) != unsafe.Offsetof(record.After) || record.Before != 17 || record.After != 51 {
		panic("zero-size generic field assignment")
	}
	if boxed := any(Choose(Tick[T](), value)); boxed == nil {
		panic("zero-size generic argument")
	}
	if Variadic(Tick[T](), Tick[T]()) != 2 {
		panic("zero-size variadic generic arguments")
	}
	values := make([][0]T, 3, 5)
	if values == nil || len(values) != 3 || cap(values) != 5 {
		panic("zero-size make")
	}
	values = append(values, Tick[T]())
	values = append(values, [][0]T{Tick[T](), Tick[T]()}...)
	if len(values) != 6 || cap(values) < 6 {
		panic("zero-size generic append")
	}
	part := values[2:4:5]
	if len(part) != 2 || cap(part) != 3 || uintptr(unsafe.Pointer(&part[0])) != uintptr(unsafe.Pointer(&values[0])) {
		panic("zero-size generic subslice")
	}
	if copy(part, values) != 2 {
		panic("zero-size generic copy")
	}
	var array [1024][0]T
	if unsafe.Sizeof(array) != 0 || len(array) != 1024 || uintptr(unsafe.Pointer(&array[0])) != uintptr(unsafe.Pointer(&array[1023])) {
		panic("zero-size array materialization")
	}
	if new([0]T) == nil {
		panic("nil zero-size pointer")
	}
	mapping := map[Empty][0]T{Empty{}: Tick[T]()}
	if _, ok := mapping[Empty{}]; !ok || len(mapping) != 1 {
		panic("zero-size map entry")
	}
	if _, ok := any(Tick[T]()).([0]T); !ok {
		panic("zero-size boxed generic identity")
	}
	callback := func(empty [0]T, arg T) T { called++; return arg }
	if any(callback(Tick[T](), value)) != any(value) {
		panic("zero-size callback argument")
	}
	var receiver Receiver[T]
	method := receiver.Value
	if any(method(value)) != any(value) {
		panic("zero-size bound receiver")
	}
	defer func(empty [0]T) { called++ }(Tick[T]())
}

func main() {
	Check(42)
	Check("value")
	if !MakePanics[Empty](-1, 0) || !MakePanics[Empty](1, 0) || !MakePanics[Empty](0, -1) {
		panic("zero-size make bounds")
	}
	if !MakePanics[int](-1, 0) || !MakePanics[int](1, 0) || !MakePanics[int](0, -1) {
		panic("ordinary make bounds")
	}
	if MakePanics[Empty](0, 0) || MakePanics[int](1, 2) {
		panic("valid make panicked")
	}
	if called != 28 {
		panic("zero-size evaluation count")
	}
	print("PASS\n")
}
