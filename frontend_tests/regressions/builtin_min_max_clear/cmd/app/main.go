package main

type namedInt int64

type item struct {
	value int
	name  string
}

type selectorValue struct{}

func (selectorValue) min() int { return 41 }

var calls int

func next(value int, want int) int {
	if calls != want {
		return -100
	}
	calls++
	return value
}

func localMinIsVisible() bool {
	min := func(left int, right int) int { return left + right }
	return min(2, 3) == 5
}

const smallest = min(9, 3, 7)
const earliest = min("z", "alpha", "middle")
const constantBase = 4
const mixedConstant = max(1, 2.0, 10)
const shiftedConstant = max(1<<40, 7)
const hexadecimalConstant = min(0x1.8p+1, 4.0)
const typedConstant namedInt = 9
const namedConstant = min(typedConstant, 4)

func main() {
	selector := selectorValue{}
	if selector.min() != 41 || !localMinIsVisible() {
		print("FAIL")
		return
	}
	if min(next(7, 0), next(2, 1), next(5, 2)) != 2 || max(next(7, 3), next(2, 4), next(5, 5)) != 7 || calls != 6 {
		print("FAIL")
		return
	}
	if smallest != 3 || earliest != "alpha" || mixedConstant != 10.0 || shiftedConstant != 1<<40 || hexadecimalConstant != 3.0 || namedConstant != namedInt(4) || min(constantBase, 8) != 4 || min("zoo", "apple", "middle") != "apple" || max("zoo", "apple", "middle") != "zoo" {
		print("FAIL")
		return
	}
	if min(int8(3), int8(-4)) != int8(-4) || max(uint8(3), uint8(9)) != uint8(9) || min(int16(300), int16(-20)) != int16(-20) || max(uint16(300), uint16(20)) != uint16(300) {
		print("FAIL")
		return
	}
	if min(int32(70000), int32(-2)) != int32(-2) || max(uint32(70000), uint32(2)) != uint32(70000) || min(int64(1<<40), int64(7)) != int64(7) || max(uint64(1<<40), uint64(7)) != uint64(1<<40) {
		print("FAIL")
		return
	}
	var intLeft int = 8
	var intRight int = -3
	var uintLeft uint = 8
	var uintRight uint = 3
	var uintptrLeft uintptr = 8
	var uintptrRight uintptr = 3
	if min(intLeft, intRight) != intRight || max(uintLeft, uintRight) != uintLeft || max(uintptrLeft, uintptrRight) != uintptrLeft {
		print("FAIL")
		return
	}
	var floatLeft32 float32 = 7.5
	var floatRight32 float32 = -2.25
	var floatLeft64 float64 = 11.5
	var floatRight64 float64 = 4.25
	if min(floatLeft32, floatRight32) != floatRight32 || max(floatLeft64, floatRight64) != floatLeft64 {
		print("FAIL")
		return
	}
	left := namedInt(12)
	right := namedInt(5)
	var named namedInt = min(left, right)
	if named != right {
		print("FAIL")
		return
	}
	values := []item{{value: 1, name: "one"}, {value: 2, name: "two"}}
	clear(values)
	if values[0].value != 0 || values[0].name != "" || values[1].value != 0 || values[1].name != "" || len(values) != 2 {
		print("FAIL")
		return
	}
	entries := map[string]int{"one": 1, "two": 2}
	clear(entries)
	if len(entries) != 0 {
		print("FAIL")
		return
	}
	var nilSlice []int
	var nilMap map[string]int
	clear(nilSlice)
	clear(nilMap)
	print("PASS\n")
}
