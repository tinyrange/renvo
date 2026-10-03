package model

type Int = int
type Byte = byte
type Rune = rune
type Callback[T any] func(T) T
type Alias[T any] = func(T) T

func Convert[T any](f Callback[T]) func(T) T { return (func(T) T)(f) }
func Nil[T any]() func(T) T                  { return (func(T) T)(nil) }
func Box[T any](v T) any                     { return v }
func Read[T any](v any) (T, bool)            { v2, ok := v.(T); return v2, ok }
func New(delta Int) Callback[Int] {
	return Callback[Int](func(v Int) Int { return v + delta })
}
func Pointer[T any](v T) func() *T { return func() *T { return &v } }
