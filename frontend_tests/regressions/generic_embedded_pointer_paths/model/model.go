package model

type Leaf[T any] struct {
	Padding int
	Value   T
}

type Middle[T any] struct {
	Padding int
	*Leaf[T]
}

type ValueMiddle[T any] struct {
	Padding string
	Middle[T]
}

type Outer[T any] struct {
	Padding int
	*ValueMiddle[T]
}

type Extra[T any] struct {
	Padding int
	*Outer[T]
}

func Read[T any](v Extra[T]) T { return v.Value }

func ReadPointer[T any](v *Extra[T]) T { return v.Value }

func Set[T any](v *Extra[T], value T) { v.Value = value }

func Address[T any](v *Extra[T]) *T { return &v.Value }

func ReadSlice[T any](values []Extra[T], index int) T { return values[index].Value }

func identity[T any](v T) T { return v }

func ReadCall[T any](v Extra[T]) T { return identity(v).Value }

func ReadPointerCall[T any](v *Extra[T]) T { return identity(v).Value }

func ReadAssertion[T any](v any) T { return v.(Extra[T]).Value }

func ReadPointerAssertion[T any](v any) T { return v.(*Extra[T]).Value }

type Shadow[T ~int] struct{ Leaf[T] }

func (v Shadow[T]) Get() T { return T(77) }

func (v Leaf[T]) Get() T { return v.Value }

func ReadMethod[T interface{ Get() int }](v T) int { return v.Get() }
