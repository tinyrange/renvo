package main

import "errors"

type wrapper struct{ inner error }

func (w wrapper) Error() string { return "wrapper" }
func (w wrapper) Unwrap() error { return w.inner }

type matcher struct{ target error }

func (matcher) Error() string          { return "matcher" }
func (m matcher) Is(target error) bool { return target == m.target }

type specialError struct{ text string }

func (e *specialError) Error() string { return e.text }

type asWrapper struct{ value *specialError }

func (w asWrapper) Error() string { return "as wrapper" }
func (w asWrapper) As(target any) bool {
	output, ok := target.(**specialError)
	if !ok {
		return false
	}
	*output = w.value
	return true
}

func main() {
	first := errors.New("first")
	second := errors.New("second")
	wanted := &specialError{text: "special"}
	var found *specialError
	if !errors.As(wrapper{inner: asWrapper{value: wanted}}, &found) || found != wanted {
		print("FAIL\n")
		return
	}
	joined := errors.Join(wrapper{inner: first}, nil, second)
	if joined == nil || joined.Error() != "wrapper\nsecond" || !errors.Is(joined, first) || !errors.Is(joined, second) || !errors.Is(matcher{target: first}, first) || errors.Is(first, errors.New("first")) {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
