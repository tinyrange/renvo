package main

import "fmt"

type value struct{ text string }

func (v value) String() string { return v.text }

type problem struct{}

func (p problem) String() string { return "string" }
func (p problem) Error() string  { return "error" }

type pointer struct{ n int }

func (p *pointer) String() string {
	if p == nil {
		return "nil receiver"
	}
	return "pointer"
}
func main() {
	v := value{"hello\n"}
	if fmt.Sprintf("%s|%v|%q|%x", v, v, v, v) != "hello\n|hello\n|\"hello\\n\"|68656c6c6f0a" {
		panic("string verbs")
	}
	if fmt.Sprintf("%s %v", problem{}, problem{}) != "error error" {
		panic("error precedence")
	}
	if fmt.Sprint(value{"a"}, value{"b"}, 3) != "a b 3" {
		panic("spacing")
	}
	var p *pointer
	if fmt.Sprintf("%s %s", p, &pointer{1}) != "nil receiver pointer" {
		panic("pointer method")
	}
	println("PASS")
}
