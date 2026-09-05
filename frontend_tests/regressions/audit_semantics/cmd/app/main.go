package main

import (
	"audit/semantics/lib"
	"fmt"
)

type S struct { _ int; X int; _ string }
type sink struct { text string }
func (s *sink) Write(p []byte) (int, error) { s.text += string(p); return len(p), nil }

var order int
func first() int { order = order*10 + 1; return 3 }
func second() int { order = order*10 + 2; return 4 }

func nilCall(deferred bool) (ok bool) {
	defer func() { ok = recover() != nil }()
	var f func()
	if deferred { defer f() } else { f() }
	return
}

func nilAddress() (ok bool) {
	defer func() { ok = recover() != nil }()
	var p *int
	_ = &*p
	return
}

func main() {
	x :=
		lib.Value
	x =
		x + 1
	if x != 8 { panic("assignment newline or module import") }
	{ var x = x + 1; if x != 9 { panic("initializer scope") } }
	if !nilCall(false) || !nilCall(true) || !nilAddress() { panic("nil recovery") }
	if first() < second() {} else { panic("comparison") }
	if order != 12 { panic("evaluation order") }
	a := S{1, 2, "a"}
	b := S{9, 2, "b"}
	if a != b { panic("blank fields") }
	m := map[S]int{a: 42}
	if m[b] != 42 { panic("blank map key") }
	var left interface{} = a
	var right interface{} = b
	if left != right { panic("blank interface") }
	if fmt.Sprintln("a", 1, "b") != "a 1 b\n" || fmt.Sprintln() != "\n" { panic("Sprintln spacing") }
	var out sink
	n, err := fmt.Fprintln(&out, "a", 1, "b")
	if err != nil || n != 6 || out.text != "a 1 b\n" { panic("Fprintln spacing") }
	println("PASS")
}
