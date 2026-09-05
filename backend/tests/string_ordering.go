package main

var order int

type Text struct{ value string }

func leftText() string { order = order*10 + 1; return "aa" }
func rightText() Text  { order = order*10 + 2; return Text{value: "aa"} }
func appMain(args []string) int {
	if !("a" < "b" && "aa" > "a" && "" <= "" && "x" >= "x") {
		return 1
	}
	values := []string{"a", "b"}
	if values[0] >= values[1] {
		return 2
	}
	a := "a"
	b := "ab"
	less := a < b
	if !less || b <= a {
		return 3
	}
	if leftText() != rightText().value || order != 12 {
		return 4
	}
	order = 0
	if leftText() > rightText().value || order != 12 {
		return 5
	}
	print("PASS\n")
	return 0
}
