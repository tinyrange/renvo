package model

const Table = "abc"

type Box[T any] struct{ Value T }

func Id[T any](v T) T { return v }

func Index[T ~int](n T) byte { return "abc"[n] }

type Number int

func (v Number) Next() Number { return v + 1 }

func Next(v Box[int]) int { return v.Value + 1 }

func Lookup(i int) int { return int(Table[i]) }

func Literal(i int) []byte { return append([]byte{}, "abc"[i]) }

func Prefix(v []byte, s string) []byte { return append(v, s...) }

// Valid declarations must be checked even when they are not instantiated or
// called. The untyped floating constant has an exact integer value.
func Unused(v int) int { return v + 1.0 }

func Callback(v string) func() string {
	return func() string { return v + "!" }
}
