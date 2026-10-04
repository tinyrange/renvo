package main

type Text string

var calls int

func destination(value *Text) *Text { calls++; return value }
func suffix(value *Text) Text       { *value = "changed"; return "b" }

func appMain() int {
	value := Text("a")
	*destination(&value) += suffix(&value)
	if calls != 1 || value != "ab" {
		panic("compound pointer evaluation")
	}
	p := &value
	pp := &p
	*(*pp) += "c"
	if value != "abc" {
		panic("indirect string destination")
	}
	print("PASS\n")
	return 0
}
