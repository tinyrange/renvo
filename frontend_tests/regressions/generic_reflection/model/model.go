package model

type Count int
type CountAlias = Count
type hidden[T any] struct{ Value T }
type Public[T any] struct{ Value T }
type Alias[T any] = Public[T]

//renvo:reflect
type Box[T any] struct {
	Public[T]
	Alias[T]
	hidden[T]
	Item T `json:"item"`
}

func New[T any](value T) Box[T] {
	return Box[T]{Public: Public[T]{value}, Alias: Alias[T]{value}, hidden: hidden[T]{value}, Item: value}
}

func (b Box[T]) Hidden() T { return b.hidden.Value }

type (
	//renvo:reflect
	Grouped[T any] struct{ Value T }
	Plain[T any]   struct{ Value T }
)
