package main

type shadowedMethodBase struct{ Value int }
type shadowedMethodOuter struct{ shadowedMethodBase }

func (v shadowedMethodOuter) Read() int { return 77 }
func (v shadowedMethodBase) Read() int  { return v.Value }

func appMain() int {
	outer := shadowedMethodOuter{shadowedMethodBase{42}}
	if outer.Read() != 77 {
		return 1
	}
	if (&outer).Read() != 77 {
		return 2
	}
	bound := outer.Read
	if bound() != 77 {
		return 3
	}
	expression := shadowedMethodOuter.Read
	if expression(outer) != 77 {
		return 4
	}
	var view interface{ Read() int } = outer
	if view.Read() != 77 {
		return 5
	}
	print("PASS\n")
	return 0
}
