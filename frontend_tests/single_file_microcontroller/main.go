// Command single_file_microcontroller is the freestanding semantic smoke suite.
//
// Its only observable runtime operation is print. It deliberately avoids os,
// files, process state, floating point, and complex arithmetic so a small
// integer-only microcontroller target can run it unchanged. Success prints
// exactly "PASS\n"; failure prints one stable feature name after "FAIL: ".
package main

import "unsafe"

var initializationTrace int

func initialized(value int) int {
	initializationTrace = initializationTrace*10 + value
	return value * 10
}

var initializedOne = initialized(1)
var initializedTwo = initialized(2)

const builtinMinimum = min(9, 3, 7)
const builtinMaximum = max(-2, 8, 4)

func init() {
	initializationTrace = initializationTrace*10 + 3
}

func init() {
	initializationTrace = initializationTrace*10 + 4
}

type namedInt int32

type point struct {
	x int
	y int
}

func (p point) sum() int {
	return p.x + p.y
}

func (p *point) move(dx, dy int) {
	p.x += dx
	p.y += dy
}

type numbered interface {
	number() int
}

type number struct {
	value int
}

func (n number) number() int {
	return n.value
}

type pair struct {
	left  int
	right int
}

type layout struct {
	first byte
	value uint32
	last  byte
}

var builtinCalls int

func orderedBuiltinValue(value, expectedCall int) int {
	if builtinCalls != expectedCall {
		return -1000
	}
	builtinCalls++
	return value
}

func testInitialization() bool {
	if initializedOne != 10 || initializedTwo != 20 {
		return fail("initialization/values")
	}
	if initializationTrace != 1234 {
		return fail("initialization/order")
	}
	return true
}

func testArithmeticAndConversions() bool {
	if 17+25 != 42 || 50-8 != 42 || 6*7 != 42 || 85/2 != 42 || 85%43 != 42 {
		return fail("arithmetic/integer")
	}
	var signed int32 = -91
	var unsigned uint32 = 81
	if signed/7 != -13 || unsigned/9 != 9 {
		return fail("arithmetic/signed-unsigned")
	}
	a, b := 3, 9
	a, b = b, a
	if a != 9 || b != 3 {
		return fail("assignment/multiple")
	}

	truncateSigned := 255
	truncateUnsigned := 257
	if int8(truncateSigned) != -1 || uint8(truncateUnsigned) != 1 {
		return fail("conversions/truncation")
	}
	var i16 int16 = -1234
	var u16 uint16 = 65000
	if int32(i16) != -1234 || uint32(u16) != 65000 {
		return fail("conversions/extension")
	}
	n := namedInt(44)
	if int(n) != 44 || namedInt(int16(44)) != n {
		return fail("conversions/named")
	}
	return true
}

func testBitwiseAndComparison() bool {
	var value uint32 = 0x3c
	if value&0x0f != 0x0c || value|0x03 != 0x3f || value^0x0f != 0x33 {
		return fail("bitwise/operators")
	}
	if uint32(1)<<30 != 1073741824 {
		return fail("bitwise/left-shift")
	}
	if uint32(0x80000000)>>31 != 1 {
		return fail("bitwise/right-shift")
	}
	a, b := 5, 8
	if !(a < b && b > a && a <= 5 && b >= 8 && a != b) {
		return fail("comparison/integer")
	}
	return true
}

func testControlFlow() bool {
	sum := 0
	for i := 0; i < 10; i++ {
		if i == 2 {
			continue
		}
		if i == 8 {
			break
		}
		sum += i
	}
	if sum != 26 {
		return fail("control/for")
	}

	count := 0
	for count < 3 {
		count++
	}
	switch sum {
	case 1, 2:
		return fail("control/switch-case")
	case 26:
		count += 4
	default:
		return fail("control/switch-default")
	}
	switch {
	case count < 7:
		return fail("control/expressionless-switch")
	case count == 7:
		count++
	}
	fall := 0
	switch 1 {
	case 1:
		fall++
		fallthrough
	case 2:
		fall += 2
	}
	if fall != 3 || count != 8 {
		return fail("control/fallthrough")
	}

	iterations := 0
outer:
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			iterations++
			if i == 1 && j == 2 {
				break outer
			}
		}
	}
	if iterations != 7 {
		return fail("control/labeled-break")
	}
	goto destination
unreachable:
	return fail("control/goto")
destination:
	if false {
		goto unreachable
	}
	return true
}

func factorial(value int) int {
	if value < 2 {
		return 1
	}
	return value * factorial(value-1)
}

func quotientAndRemainder(left, right int) (int, int) {
	return left / right, left % right
}

func namedReturn(value int) (result int) {
	result = value + 2
	return
}

func variadicSum(prefix int, values ...int) int {
	total := prefix
	for _, value := range values {
		total += value
	}
	return total
}

func testFunctions() bool {
	if factorial(6) != 720 {
		return fail("functions/recursion")
	}
	quotient, remainder := quotientAndRemainder(29, 6)
	if quotient != 4 || remainder != 5 || namedReturn(40) != 42 {
		return fail("functions/results")
	}
	values := []int{2, 3, 4}
	if variadicSum(1, values...) != 10 || variadicSum(42) != 42 {
		return fail("functions/variadic")
	}
	return true
}

func testArraysAndSlices() bool {
	values := [5]int{1, 2, 3, 4, 5}
	copyValue := values
	copyValue[0] = 9
	if values[0] != 1 || copyValue[0] != 9 || len(values) != 5 {
		return fail("arrays/value-copy")
	}
	matrix := [2][3]int{{1, 2, 3}, {4, 5, 6}}
	if matrix[1][2] != 6 {
		return fail("arrays/nested")
	}

	slice := make([]int, 2, 4)
	slice[0], slice[1] = 1, 2
	slice = append(slice, 3, 4)
	if len(slice) != 4 || cap(slice) != 4 || slice[3] != 4 {
		return fail("slices/make-append")
	}
	slice = append(slice, 5)
	if len(slice) != 5 || cap(slice) < 5 || slice[4] != 5 {
		return fail("slices/growth")
	}
	destination := make([]int, 5)
	if copy(destination, slice) != 5 || destination[2] != 3 {
		return fail("slices/copy")
	}
	copy(destination[1:], destination[:4])
	if destination[0] != 1 || destination[1] != 1 || destination[4] != 4 {
		return fail("slices/overlap")
	}
	limited := slice[1:3:3]
	limited = append(limited, 99)
	if limited[2] != 99 || slice[3] != 4 {
		return fail("slices/full-expression")
	}
	var nilSlice []int
	if nilSlice != nil || len(nilSlice) != 0 || cap(nilSlice) != 0 {
		return fail("slices/nil")
	}
	return true
}

func testStringsAndRunes() bool {
	text := "Renvo"
	if len(text) != 5 || text[0] != 'R' || text[4] != 'o' {
		return fail("strings/basic")
	}
	return true
}

func testStructsPointersAndMethods() bool {
	p := point{x: 10, y: 20}
	if p.sum() != 30 {
		return fail("structs/method")
	}
	p.move(3, -4)
	if p != (point{13, 16}) || p.sum() != 29 {
		return fail("pointers/receiver")
	}
	alias := &p
	alias.x = 21
	if p.x != 21 || alias.y != 16 {
		return fail("pointers/alias")
	}
	created := new(pair)
	created.left, created.right = 19, 23
	if *created != (pair{left: 19, right: 23}) {
		return fail("pointers/new")
	}
	return true
}

func apply(value int, operation func(int) int) int {
	return operation(value)
}

func testClosuresAndFunctionValues() bool {
	base := 10
	add := func(value int) int {
		base += value
		return base
	}
	if add(2) != 12 || add(3) != 15 || base != 15 {
		return fail("closures/capture")
	}
	double := func(value int) int { return value * 2 }
	if apply(21, double) != 42 {
		return fail("functions/value")
	}
	var absent func()
	if absent != nil {
		return fail("functions/nil")
	}
	return true
}

func interfaceNumber(value numbered) int {
	return value.number()
}

func testInterfaces() bool {
	var dynamic numbered = number{value: 42}
	if dynamic.number() != 42 || interfaceNumber(dynamic) != 42 {
		return fail("interfaces/dispatch")
	}
	return true
}

func testUnsafeAndBuiltins() bool {
	var value layout
	if unsafe.Sizeof(value.first) != 1 || unsafe.Sizeof(value) < 9 {
		return fail("unsafe/layout")
	}
	original := pair{left: 19, right: 23}
	raw := unsafe.Pointer(&original)
	converted := (*pair)(raw)
	if converted.left != 19 || converted.right != 23 {
		return fail("unsafe/pointer-conversion")
	}

	if builtinMinimum != 3 || builtinMaximum != 8 {
		return fail("builtins/min-max-constant")
	}
	builtinCalls = 0
	minimum := min(
		orderedBuiltinValue(7, 0),
		orderedBuiltinValue(2, 1),
		orderedBuiltinValue(5, 2),
	)
	maximum := max(
		orderedBuiltinValue(7, 3),
		orderedBuiltinValue(2, 4),
		orderedBuiltinValue(5, 5),
	)
	if minimum != 2 || maximum != 7 || builtinCalls != 6 {
		return fail("builtins/min-max-evaluation")
	}
	values := []pair{{left: 1, right: 2}, {left: 3, right: 4}}
	clear(values)
	if len(values) != 2 || values[0] != (pair{}) || values[1] != (pair{}) {
		return fail("builtins/clear")
	}
	return true
}

func fail(name string) bool {
	print("FAIL: ")
	print(name)
	print("\n")
	return false
}

func main() {
	if !testInitialization() ||
		!testArithmeticAndConversions() ||
		!testBitwiseAndComparison() ||
		!testControlFlow() ||
		!testFunctions() ||
		!testArraysAndSlices() ||
		!testStringsAndRunes() ||
		!testStructsPointersAndMethods() ||
		!testClosuresAndFunctionValues() ||
		!testInterfaces() ||
		!testUnsafeAndBuiltins() {
		return
	}
	print("PASS\n")
}
