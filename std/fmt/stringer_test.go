package fmt

import "testing"

type stringerValue struct{ text string }

func (v stringerValue) String() string { return v.text }

type stringerError struct{}

func (v stringerError) String() string { return "string" }
func (v stringerError) Error() string  { return "error" }

func TestStringMethods(t *testing.T) {
	v := stringerValue{"hello\n"}
	if got := Sprintf("%s|%v|%q|%x", v, v, v, v); got != "hello\n|hello\n|\"hello\\n\"|68656c6c6f0a" {
		t.Fatal(got)
	}
	if got := Sprintf("%s %v", stringerError{}, stringerError{}); got != "error error" {
		t.Fatal(got)
	}
	if got := Sprint(stringerValue{"a"}, stringerValue{"b"}, 3); got != "a b 3" {
		t.Fatal(got)
	}
}
