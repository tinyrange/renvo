package main

type AliasMemberScalar = int
type AliasMemberArg = struct{ Value AliasMemberScalar }
type AliasMemberOther = struct{ Value int }
type AliasMemberBase interface{ First() int }
type AliasMemberReader = interface {
	AliasMemberBase
	Second() int
}
type AliasMemberFlat = interface {
	Second() int
	First() int
}
type AliasMemberValue int

const AliasMemberLeftLength = 2
const AliasMemberRightLength = 1 + 1

type AliasMemberArrayArg = struct{ Value [AliasMemberLeftLength]int }
type AliasMemberArrayOther = struct{ Value [AliasMemberRightLength]int }

func (v AliasMemberValue) First() int                                        { return int(v) }
func (v AliasMemberValue) Second() int                                       { return 0 }
func aliasMemberApply(f func(AliasMemberArg) int) int                        { return f(AliasMemberArg{42}) }
func aliasMemberRead(f func(AliasMemberReader) int, v AliasMemberReader) int { return f(v) }
func aliasMemberArray(f func(AliasMemberArrayArg) int) int {
	return f(AliasMemberArrayArg{[AliasMemberLeftLength]int{20, 22}})
}

func appMain() int {
	var f func(func(AliasMemberOther) int) int = aliasMemberApply
	if f(func(v AliasMemberOther) int { return v.Value }) != 42 {
		return 1
	}
	var r func(func(AliasMemberFlat) int, AliasMemberFlat) int = aliasMemberRead
	if r(func(v AliasMemberFlat) int { return v.First() + v.Second() }, AliasMemberValue(42)) != 42 {
		return 2
	}
	var array func(func(AliasMemberArrayOther) int) int = aliasMemberArray
	if array(func(v AliasMemberArrayOther) int { return v.Value[0] + v.Value[1] }) != 42 {
		return 3
	}
	print("PASS\n")
	return 0
}
