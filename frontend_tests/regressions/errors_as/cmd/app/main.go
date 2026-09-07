package main

import "errors"

type detail struct { n int }
func (e *detail) Error() string { return "detail" }

func main() {
	want := &detail{n:3}
	var got *detail
	if !errors.As(want, &got) || got != want { panic("As") }
	print("PASS\n")
}
