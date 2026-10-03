package main

import (
	"example.com/genericfunctions/box"
	"fmt"
)

type Node[T any] struct {
	Value T
	Next  *Node[T]
}

func identity[T any](value T) T                        { return value }
func send[C ~chan E | ~chan<- E, E any](ch C, value E) { ch <- value }
func receive[C ~chan E | ~<-chan E, E any](ch C) E     { return <-ch }
func tryReceive[T any](ch <-chan T) (T, bool) {
	select {
	case value := <-ch:
		return value, true
	default:
		var zero T
		return zero, false
	}
}
func double[T ~int](value T) T            { return value + value }
func apply[T any](f func(T) T, value T) T { return f(value) }
func copySlice[S ~[]E, E any](s S) S      { return s }
func outerCopy[S ~[]E, E any](s S) S      { return copySlice(s) }
func pair[T any](v T) (T, T)              { return v, v }
func forwardPair[T any](v T) (T, T)       { return pair(v) }
func getIdentity[T any]() func(T) T       { return identity }
func capturedIdentity[T any](value T) func() T {
	return func() T { return identity(value) }
}
func closureIdentity[T any]() func(T) T {
	return func(value T) T { var zero T; _ = zero; return identity(value) }
}
func localClosure[T any](value T) func() T {
	return func() T { type Box struct{ Value T }; b := Box{value}; return identity(b).Value }
}
func outerLocalClosure[T any](value T) func() T {
	type Box struct{ Value T }
	return func() T { var b Box; b.Value = value; return b.Value }
}
func useIdentity(f func(int) int) int { return f(42) }
func second[A, B any](a A, b B) B     { return b }
func useSecond[A, B any](f func(A, B) B, v B) B {
	var zero A
	return f(zero, v)
}
func localBox[T any](value T) T {
	type Box struct {
		Value T
		Next  *Box
	}
	b := Box{Value: value}
	b.Next = &b
	return identity(b).Next.Value
}

func keys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	return out
}
func lookup[M ~map[K]V, K comparable, V any](m M, key K) (V, bool) {
	value, ok := m[key]
	return value, ok
}
func asserted[T any](value any) (T, bool) {
	v, ok := value.(T)
	return v, ok
}
func mapSlice[S ~[]E, E any, U any](s S, f func(E) U) []U {
	out := make([]U, 0, len(s))
	for _, v := range s {
		out = append(out, f(v))
	}
	return out
}

type Number int

type Wrapped[T any] struct{ box.Box[T] }
type Outer[T any] struct{ *Wrapped[T] }
type EmbeddedAlias[T any] = box.Box[T]
type FixedBox = box.Box[int]
type FixedHolder struct{ FixedBox }
type AliasedHolder[T any] struct{ EmbeddedAlias[T] }
type IntFunction func(int) int
type hiddenValue int

func (v hiddenValue) hidden() int { return int(v) }
func classify[T any](value T) int {
	switch v := any(value).(type) {
	case int:
		return v
	case string:
		return len(v)
	case T:
		_ = v
		return 7
	default:
		return 0
	}
}
func constantLength[T ~int]() T { const prefix string = "ab"; return len(prefix + "c") }

const (
	ordinalZero = iota
	ordinalOne
	ordinalTwo
)

func globalOrdinal[T ~int]() T { return ordinalTwo }

func roundedConstant[T ~int8]() T                      { return T(float32(127.000001)) }
func constantParts[T ~int]() T                         { return real(complex(40, 3)) + imag(2i) }
func typedMinimum[T ~int8]() T                         { return T(min(int(127), 128)) }
func complementConstant[T ~uint8]() T                  { return T(^uint8(1)) }
func comparisonBoolean[T ~bool](v int) T               { return v == 0 }
func interfaceEqual[T comparable](v T, other any) bool { return v == other }
func inferredArray[T any](v T) [3]T                    { return [...]T{2: v} }
func shiftedConstant[T ~int]() T                       { return (1 << 100) >> 99 }

func ordinaryLocalClosure() func() int {
	type Local struct{ Value int }
	return func() int { v := Local{42}; return identity(v).Value }
}

func readBox[T interface{ Get() E }, E any](value T) E { return value.Get() }

func sparse[T any](v T) []T                        { return []T{2: v, 0: v, v} }
func literalPointer[T any](v T) *struct{ Value T } { return &(struct{ Value T }{v}) }
func arrayCopy[S ~[]E, E any](v S) [2]E            { return [2]E(v) }
func arrayView[S ~[]E, E any](v S) *[2]E           { return (*[2]E)(v) }
func constantLetters[T ~int]() T                   { return len(string(65)) + len(string(0x1f600)) + len(string(-1)) }
func retag[T ~struct {
	Value int "old"
}](v T) struct {
	Value int "new"
} {
	return struct {
		Value int "new"
	}(v)
}

func contextualShift[T ~int](n uint) T      { return 1.0 << n }
func fullSlice[S ~[]E, E any](s S, n int) S { return s[:n:2] }
func stringSlice[T ~string]() T             { return T("abc"[1:]) }
func reusedVariable[T any](v T) T           { v, x := v, 1; _ = x; return v }
func nativeComplement[T ~uint]() T          { return T(^uint(0)) }
func pointerComplement[T ~uintptr]() T      { return T(^uintptr(0)) }
func parenthesizedLength[T ~int]() T        { var a [3]int; const n = len((a)); return T(n) }
func convertedLength[T ~int]() T            { const n = cap(([3]int)([3]int{int(42)})); return T(n) }
func hugeComparison[T ~bool]() T            { return T((1<<100) > (1<<99) && !false && 1i == complex(0, 1)) }

func main() {
	if nativeComplement[uint]() != ^uint(0) || pointerComplement[uintptr]() != ^uintptr(0) {
		panic("generic target width")
	}
	if parenthesizedLength[int]() != 3 || convertedLength[int]() != 3 || !hugeComparison[bool]() {
		panic("generic constant expression")
	}
	if contextualShift[int](3) != 8 || stringSlice[string]() != "bc" || reusedVariable(42) != 42 {
		panic("generic contextual expression")
	}
	bounded := fullSlice([]int{42, 7, 3}, 1)
	if len(bounded) != 1 || cap(bounded) != 2 || bounded[0] != 42 {
		panic("generic full slice")
	}

	if !interfaceEqual(42, any(42)) || interfaceEqual(42, any("42")) || inferredArray(42)[2] != 42 || shiftedConstant[int]() != 2 {
		panic("generic operators or inferred arrays")
	}
	sparseValues := sparse(42)
	if len(sparseValues) != 3 || sparseValues[0] != 42 || sparseValues[1] != 42 || sparseValues[2] != 42 || literalPointer(42).Value != 42 || constantLetters[int]() != 8 || retag(struct {
		Value int "old"
	}{42}).Value != 42 {
		panic("generic composite or conversion")
	}
	arraySource := []int{1, 2}
	arrayValue := arrayCopy(arraySource)
	arrayPointer := arrayView(arraySource)
	arraySource[0] = 42
	if arrayValue[0] != 1 || arrayPointer[0] != 42 {
		panic("generic slice to array conversion")
	}
	if roundedConstant[int8]() != 127 || constantParts[int]() != 42 || typedMinimum[int8]() != 127 || complementConstant[uint8]() != 254 || !comparisonBoolean[bool](0) {
		panic("generic constant semantics")
	}
	var mixed interface {
		box.Hidden
		hidden() int
	}
	if box.Identity(mixed) != nil {
		panic("mixed private interface identity")
	}
	if ordinaryLocalClosure()() != 42 {
		panic("ordinary closure local type")
	}
	if globalOrdinal[int]() != 2 {
		panic("generic global constant")
	}
	var hidden interface{ hidden() int } = hiddenValue(42)
	if box.Identity(hidden).hidden() != 42 || classify(42) != 42 || classify("yes") != 3 || classify(1.5) != 7 || constantLength[int]() != 3 {
		panic("generic interface or constant")
	}
	ch := make(chan int, 1)
	send(ch, 42)
	if receive(ch) != 42 {
		panic("generic channel")
	}
	channelValue, ready := tryReceive(ch)
	if ready || channelValue != 0 {
		panic("generic select default")
	}
	if outerLocalClosure(42)() != 42 || outerLocalClosure("yes")() != "yes" {
		panic("closure local type capture")
	}
	type Local int
	if box.Identity(Local(42)) != Local(42) || localBox(42) != 42 || localBox("yes") != "yes" {
		panic("generic local type identity")
	}
	w := Wrapped[int]{Box: box.Box[int]{Value: 42}}
	fixed := FixedHolder{FixedBox: box.Box[int]{Value: 42}}
	aliased := AliasedHolder[int]{EmbeddedAlias: box.Box[int]{Value: 42}}
	if fixed.FixedBox.Value != 42 || aliased.EmbeddedAlias.Value != 42 {
		panic("embedded generic alias")
	}
	original := struct{ box.Box[int] }{box.Box[int]{Value: 42}}
	_, sameOriginal := asserted[struct{ box.Box[int] }](any(original))
	_, sameAlias := asserted[struct{ EmbeddedAlias[int] }](any(original))
	if !sameOriginal || sameAlias {
		panic("anonymous embedded field identity")
	}
	o := Outer[int]{&w}
	if readBox(w) != 42 || w.Box.Value != 42 || identity(o.Value) != 42 || o.Box.Value != 42 {
		panic("generic embedded field or promoted method")
	}
	value := identity[int](21)
	var node Node[int]
	node.Value = double(value)
	if identity(node.Value) != 42 || identity("hello") != "hello" {
		panic("generic function result")
	}
	var b box.Box[Number]
	b.Set(Number(42))
	if identity(b.Get()) != Number(42) || box.Identity(Number(7)) != Number(7) {
		panic("generic receiver or cross-package result")
	}
	var f func(int) int = identity
	var imported func(string) string = box.Identity
	f = identity
	if f(9) != 9 || imported("yes") != "yes" || apply(identity, 42) != 42 || len(outerCopy([]int{1, 2})) != 2 {
		panic("generic inference")
	}
	if useIdentity(identity) != 42 || getIdentity[int]()(42) != 42 || useSecond(second[int], 42) != 42 {
		panic("generic function argument and return inference")
	}
	if capturedIdentity(42)() != 42 || capturedIdentity("yes")() != "yes" || closureIdentity[int]()(42) != 42 {
		panic("generic closure")
	}
	var named IntFunction = identity
	callbacks := []func(int) int{identity}
	use := func(f IntFunction) int { return f(42) }
	if callbacks[0](42) != 42 || named(42) != 42 || use(identity) != 42 || localClosure(42)() != 42 || localClosure("yes")() != "yes" {
		panic("generic closure local type or function variable")
	}
	a, c := forwardPair(21)
	if identity(a)+identity(c) != 42 {
		panic("generic multiple results")
	}
	key := keys(map[string]int{"answer": 42})
	found, present := lookup(map[string]int{"answer": 42}, "answer")
	assertion, matches := asserted[int](any(42))
	if found != 42 || !present || assertion != 42 || !matches {
		panic("generic comma-ok inference")
	}
	mapped := mapSlice([]int{20, 21}, func(v int) int { return v * 2 })
	if len(key) != 1 || key[0] != "answer" || len(mapped) != 2 || mapped[1] != 42 {
		panic("generic collections")
	}
	fmt.Println("PASS")
}
