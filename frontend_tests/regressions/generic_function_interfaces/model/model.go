package model

type Callback[T any] func(T) T

type Alias[T any] = Callback[T]

type Other[T any] func(T) T

type Unnamed[T any] = func(T) T

func ToUnnamed(f Callback[int]) func(int) int { return f }

func ToDefined(f func(int) int) Callback[int] { return f }

func NewOther(delta int) Other[int] {
	return Other[int](func(v int) int { return v + delta })
}

func (f Callback[T]) Apply(v T) T { return f(v) }

func New(delta int) Callback[int] {
	return Callback[int](func(v int) int { return v + delta })
}

func Box[T any](v T) any { return v }

func Equal[T comparable](a, b T) bool { return a == b }

func Read[T any](v any) (T, bool) {
	x, ok := v.(T)
	return x, ok
}

func Put[K comparable, V any](m map[K]V, key K, value V) { m[key] = value }

func Lookup[K comparable, V any](m map[K]V, key K) V { return m[key] }

func Delete[K comparable, V any](m map[K]V, key K) { delete(m, key) }
