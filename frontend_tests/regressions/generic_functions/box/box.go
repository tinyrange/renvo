package box

type Box[T any] struct{ Value T }

func (b Box[Element]) Get() Element       { return b.Value }
func (b *Box[Element]) Set(value Element) { b.Value = value }
func Identity[T any](value T) T           { return value }

type Hidden interface{ boxed() int }
