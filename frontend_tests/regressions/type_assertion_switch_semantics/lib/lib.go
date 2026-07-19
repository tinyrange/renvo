package lib

type Number interface {
	Number() int
}

type EmbeddedNumber interface {
	Number
}

type Item struct {
	Value int
}

func (value Item) Number() int {
	return value.Value
}

type Pointer struct {
	Value int
}

func (value *Pointer) Number() int {
	return value.Value
}

type Wrong struct{}

func (Wrong) Number(extra int) int {
	return extra
}

var Calls int

func Choose(which int) interface{} {
	Calls++
	if which == 1 {
		return Item{Value: 9}
	}
	if which == 2 {
		return "text"
	}
	if which == 4 {
		return &Pointer{Value: 11}
	}
	if which == 5 {
		return Pointer{Value: 12}
	}
	if which == 6 {
		return Wrong{}
	}
	return nil
}
