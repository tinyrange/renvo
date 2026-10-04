package main

import (
	"example.com/genericcallablelayout/lib"
	"unsafe"
)

var calls int

func Factory[T any](value T) func(T) T {
	calls++
	return func(other T) T { return value }
}

func Check[T comparable](value, changed T) {
	fn := lib.Identity[T]
	const n = unsafe.Sizeof(lib.Identity[T])
	if unsafe.Sizeof(fn) != n {
		panic("direct function and callback layouts disagree")
	}
	var array [n]byte
	if _, ok := any(array).([unsafe.Sizeof(fn)]byte); !ok {
		panic("callback array bound identity")
	}
	if _, ok := any(array).(lib.Bytes[T]); ok {
		panic("defined array identity lost")
	}
	if len(lib.Bytes[T](array)) != int(n) {
		panic("cross-package function layout")
	}
	const aligned = unsafe.Alignof(fn)
	if aligned != unsafe.Alignof(lib.Identity[T]) {
		panic("callable alignment")
	}
	receiver := lib.Receiver[T]{Value: value}
	fn = receiver.Get
	receiver.Value = changed
	if fn(changed) != value || unsafe.Sizeof(fn) != n || unsafe.Sizeof(receiver.Get) != n {
		panic("bound receiver snapshot or layout")
	}
	pointer := &lib.Pointer[T]{Value: value}
	fn = pointer.Get
	pointer.Value = changed
	if fn(value) != changed || unsafe.Sizeof(fn) != n {
		panic("bound pointer receiver")
	}
	var defined lib.Defined[T] = Factory(value)
	fn = defined
	if fn(changed) != value || unsafe.Sizeof(defined) != n {
		panic("defined callable conversion")
	}
	var record struct {
		Before int
		Fn     func(T) T
		After  int
	}
	record.Before, record.Fn, record.After = 17, fn, 51
	const offset = unsafe.Offsetof(record.After)
	if offset-unsafe.Offsetof(record.Fn) != n || record.Fn(changed) != value || record.Before != 17 || record.After != 51 {
		panic("callable containing aggregate")
	}
	const factorySize = unsafe.Sizeof(Factory(value))
	if factorySize != n || calls != 1 {
		panic("layout query evaluated function operand")
	}
	fn = nil
	if fn != nil || unsafe.Sizeof(fn) != n {
		panic("nil callable layout")
	}
	calls = 0
}

func main() {
	Check(42, 73)
	Check("original", "changed")
	print("PASS\n")
}
