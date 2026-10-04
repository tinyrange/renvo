package main

import (
	"example.com/genericunsafe/bridge"
	"example.com/genericunsafe/shadow"
	"unsafe"
)

type raw unsafe.Pointer
type address uintptr

func toAddress[P ~unsafe.Pointer](value P) uintptr { return uintptr(value) }
func toPointer[P ~uintptr](value P) unsafe.Pointer { return unsafe.Pointer(value) }
func round[P ~*int | ~*byte](value P) P            { return P(unsafe.Pointer(value)) }

func alignment[T any](value T) uintptr { return unsafe.Alignof(value) }

func main() {
	value := 42
	pointer := raw(unsafe.Pointer(&value))
	integer := address(toAddress(bridge.Identity(pointer)))
	if integer == 0 || *(*int)(toPointer(integer)) != 42 {
		panic("unsafe conversion")
	}
	if round(&value) != &value || *round(&value) != 42 {
		panic("integer pointer")
	}
	var octet byte = 51
	if round(&octet) != &octet || *round(&octet) != 51 {
		panic("byte pointer")
	}
	if shadow.Sum[int]() != 34 {
		panic("predeclared shadow")
	}
	if alignment(value) < 1 || alignment(octet) != 1 {
		panic("unsafe alignment")
	}
	print("PASS\n")
}
