package main

type MemberArg = struct{ A, B int }
type MemberOther = struct {
	A int
	B int
}

func memberApply(f func(MemberArg) int) int { return f(MemberArg{20, 22}) }

func appMain() int {
	var f func(func(MemberOther) int) int = memberApply
	if f(func(v MemberOther) int { return v.A + v.B }) != 42 {
		return 1
	}
	print("PASS\n")
	return 0
}
