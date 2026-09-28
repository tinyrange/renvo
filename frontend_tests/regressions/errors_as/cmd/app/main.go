package main

import "errors"

type detail struct{ n int }

func (e *detail) Error() string { return "detail" }
func (e *detail) Number() int   { return e.n }

type numbered interface{ Number() int }
type valueError int

func (e valueError) Error() string { return "value" }

type wrapper struct{ inner error }

func (e wrapper) Error() string { return "wrapper" }
func (e wrapper) Unwrap() error { return e.inner }

type tree struct{ children []error }

func (e tree) Error() string   { return "tree" }
func (e tree) Unwrap() []error { return e.children }

type custom struct{ detail *detail }

func (e custom) Error() string { return "custom" }
func (e custom) As(target any) bool {
	if pointer, ok := target.(**detail); ok {
		*pointer = e.detail
		return true
	}
	return false
}

func invalidTarget(target any) (panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	errors.As(&detail{}, target)
	return
}

func main() {
	first := &detail{n: 3}
	second := &detail{n: 7}
	var got *detail
	match := errors.As
	if !match(first, &got) || got != first {
		panic("function value")
	}
	if !errors.As(wrapper{inner: first}, &got) || got != first {
		panic("single unwrap")
	}
	if !errors.As(tree{children: []error{nil, wrapper{inner: first}, second}}, &got) || got != first {
		panic("depth-first traversal")
	}
	if !errors.As(custom{detail: second}, &got) || got != second {
		panic("custom As")
	}
	if errors.As(errors.New("unmatched"), &got) || got != second {
		panic("nonmatch changed target")
	}
	var nilDetail *detail
	if !errors.As(nilDetail, &got) || got != nil {
		panic("typed nil")
	}
	var generic any
	if !errors.As(first, &generic) || generic != first {
		panic("any target")
	}
	var plain error
	if !errors.As(first, &plain) || plain != first {
		panic("error target")
	}
	var number numbered
	if !errors.As(first, &number) || number.Number() != 3 {
		panic("interface target")
	}
	var value valueError
	if !errors.As(valueError(9), &value) || value != 9 {
		panic("value target")
	}
	if errors.As(nil, nil) {
		panic("nil error")
	}
	var nilTarget **detail
	var invalid int
	if !invalidTarget(nil) || !invalidTarget(nilTarget) || !invalidTarget(3) || !invalidTarget(&invalid) {
		panic("invalid target accepted")
	}
	if !errors.Is(wrapper{inner: first}, first) {
		panic("Is unwrap")
	}
	print("PASS\n")
}
