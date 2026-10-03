package model

type Value int

func (v Value) Get() int               { return int(v) }
func Id[T any](int T) T                { return int }
func Pair[T any](int, string T) (T, T) { return int, string }
func Result[T any](v T) (int T)        { return v }
func Named[T any](Value T) T           { return Value }

type Box[T any] struct{ Value T }

func (int Box[T]) Get() T             { return int.Value }
func (int Box[T]) Closure() func() T  { return func() T { return int.Value } }
func Captured[T any](int T) func(T) T { return func(v T) T { return v } }
func Local[T any](v T) func() T       { int := v; return func() T { return int } }
func LocalType[T any](v T) func() T   { type int string; _ = int(""); return func() T { return v } }
func NamedLocalType[T any](v T) func() T {
	type Value string
	_ = Value("")
	return func() T { return v }
}
func Parameter[T any](v T) func(T) T           { return func(int T) T { return int } }
func ResultClosure[T any](v T) func() T        { return func() (int T) { return v } }
func CapturedResult[T any](v T) (int func() T) { return func() T { return v } }
func LocalAlias[T any](int T) T                { type U = T; var v U = int; return v }
func AliasClosure[T any](int T) func() T       { type U = T; return func() U { return int } }
func SliceClosure[T any](v T) func() []T       { int := v; return func() []T { return []T{int} } }
func Map[T any](int T) map[string]T            { return map[string]T{"key": int} }

type Alias[T any] = map[string]T

func AliasMap[T any](int T) Alias[T]                  { return Alias[T]{"key": int} }
func Read[T interface{ Get() int }](int T) func() int { return int.Get }

func Convert[T ~func(int) int](int T) T { return T(int) }
