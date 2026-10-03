package lib

import "unsafe"

type Callback[T any] = func(T) T
type Defined[T any] func(T) T
type Bytes[T any] [unsafe.Sizeof((func(T) T)(nil))]byte

func Identity[T any](value T) T { return value }

type Receiver[T any] struct {
	Value T
	Extra [5]int
}

func (r Receiver[T]) Get(value T) T { return r.Value }

type Pointer[T any] struct{ Value T }

func (r *Pointer[T]) Get(value T) T { return r.Value }
