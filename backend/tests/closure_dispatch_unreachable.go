package main

func directValue() int          { return 42 }
func unusedFactory() func() int { v := "unreachable"; return func() int { return len(v) } }
func deadLiteral() {
	v := "unreachable"
	if false {
		f := func() int { return len(v) }
		f()
	}
}
func appMain() int {
	var f func() int = directValue
	if f() != 42 {
		return 1
	}
	deadLiteral()
	if f() != 42 {
		return 2
	}
	print("PASS\n")
	return 0
}
