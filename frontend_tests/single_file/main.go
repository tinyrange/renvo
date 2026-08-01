// Command single_file is a compact end-to-end semantic suite for Renvo.
//
// It intentionally lives in one source file. A successful run prints exactly
// "PASS\n". A failure prints one stable, feature-focused name prefixed by
// "FAIL: ". Keep checks grouped by language feature so this remains useful as
// a fast smoke test rather than becoming a copy of the larger test corpora.
package main

import (
	"os"
	"unsafe"
)

var initializationTrace int

func initialized(value int) int {
	initializationTrace = initializationTrace*10 + value
	return value * 10
}

var initializedOne = initialized(1)
var initializedTwo = initialized(2)

const builtinMinimum = min(9, 3, 7)
const builtinMaximum = max(-2, 8, 4)
const builtinStringMinimum = min("zoo", "apple", "middle")
const builtinStringMaximum = max("zoo", "apple", "middle")

func init() {
	initializationTrace = initializationTrace*10 + 3
}

func init() {
	initializationTrace = initializationTrace*10 + 4
}

type namedInt int64

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

type named interface {
	name() string
}

type animal struct {
	kind string
}

func (a animal) name() string {
	return a.kind
}

type numbered interface {
	number() int
}

type baseNumber struct {
	value int
}

func (n baseNumber) number() int {
	return n.value
}

type embeddedNumber struct {
	baseNumber
	label string
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

func failure(name string) {
	print("FAIL: ")
	print(name)
	print("\n")
}

func testInitialization() string {
	if initializedOne != 10 || initializedTwo != 20 {
		return "initialization/values"
	}
	if initializationTrace != 1234 {
		return "initialization/order"
	}
	return ""
}

func testArithmetic() string {
	const large = 1 << 40
	const fraction = 7.5
	var signed int64 = -91
	var unsigned uint64 = 81
	if 17+25 != 42 || 50-8 != 42 || 6*7 != 42 || 85/2 != 42 || 85%43 != 42 {
		return "arithmetic/integer"
	}
	if signed/7 != -13 || unsigned/9 != 9 || large != 1099511627776 {
		return "arithmetic/width"
	}
	if fraction+0.5 != 8 {
		return "arithmetic/float"
	}
	var numerator float64 = 9
	floatValue := numerator / 4.0
	if floatValue != 2.25 {
		return "arithmetic/float64"
	}
	a, b := 3, 9
	a, b = b, a
	if a != 9 || b != 3 {
		return "arithmetic/multiple-assignment"
	}
	return ""
}

func testIntegerWidthsAndConversions() string {
	wideSigned := 255
	wideUnsigned := 257
	if int8(wideSigned) != -1 || uint8(wideUnsigned) != 1 {
		return "conversions/truncation"
	}
	var i16 int16 = -1234
	var u16 uint16 = 65000
	var i32 int32 = -70000
	var u32 uint32 = 4000000000
	if int64(i16) != -1234 || uint64(u16) != 65000 || int64(i32) != -70000 || uint64(u32) != 4000000000 {
		return "conversions/extension"
	}
	n := namedInt(44)
	if int(n) != 44 || namedInt(int32(44)) != n {
		return "conversions/named"
	}
	return ""
}

func testComparisonAndBoolean() string {
	a, b := 5, 8
	truth := a < b && b > a && a <= 5 && b >= 8 && a != b
	if !truth || a == b || !(true || false) || !(!false) {
		return "comparison/basic"
	}
	if "same" != "same" || "left" == "right" {
		return "comparison/string"
	}
	return ""
}

func testBitwiseAndShifts() string {
	var value uint32 = 0x3c
	if value&0x0f != 0x0c || value|0x03 != 0x3f || value^0x0f != 0x33 || value&^0x0c != 0x30 {
		return "bitwise/operators"
	}
	if uint64(1)<<40 != 1099511627776 || uint32(0x80000000)>>31 != 1 {
		return "bitwise/shifts"
	}
	var signed int8 = -1
	shifted := signed >> 3
	if shifted != -1 {
		return "bitwise/signed-shift"
	}
	return ""
}

func testControlFlow() string {
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
		return "control/for"
	}

	count := 0
	for count < 3 {
		count++
	}
	if count != 3 {
		return "control/condition-loop"
	}

	switch sum {
	case 1, 2:
		return "control/switch-wrong-case"
	case 26:
		count += 4
	default:
		return "control/switch-default"
	}
	switch {
	case count < 7:
		return "control/expressionless-switch"
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
	if fall != 3 {
		return "control/fallthrough"
	}

	outer := 0
outerLoop:
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			outer++
			if i == 1 && j == 2 {
				break outerLoop
			}
		}
	}
	if outer != 7 {
		return "control/labeled-break"
	}

	goto destination
unreachable:
	return "control/goto"
destination:
	if false {
		goto unreachable
	}
	return ""
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

func testFunctions() string {
	if factorial(6) != 720 {
		return "functions/recursion"
	}
	quotient, remainder := quotientAndRemainder(29, 6)
	if quotient != 4 || remainder != 5 || namedReturn(40) != 42 {
		return "functions/results"
	}
	values := []int{2, 3, 4}
	if variadicSum(1, values...) != 10 || variadicSum(42) != 42 {
		return "functions/variadic"
	}
	return ""
}

func testArrays() string {
	values := [5]int{1, 2, 3, 4, 5}
	copyValue := values
	copyValue[0] = 9
	if values[0] != 1 || copyValue[0] != 9 || len(values) != 5 {
		return "arrays/value-copy"
	}

	inferred := [...]string{"a", "bb", "ccc"}
	total := 0
	for index, value := range inferred {
		total += index + len(value)
	}
	if total != 9 {
		return "arrays/range"
	}

	matrix := [2][3]int{{1, 2, 3}, {4, 5, 6}}
	if matrix[1][2] != 6 {
		return "arrays/nested"
	}
	return ""
}

func testSlices() string {
	values := make([]int, 2, 4)
	values[0], values[1] = 1, 2
	values = append(values, 3, 4)
	if len(values) != 4 || cap(values) != 4 || values[3] != 4 {
		return "slices/make-append"
	}

	values = append(values, 5)
	if len(values) != 5 || cap(values) < 5 || values[0] != 1 || values[4] != 5 {
		return "slices/growth"
	}

	destination := make([]int, 5)
	if copy(destination, values) != 5 || destination[2] != 3 {
		return "slices/copy"
	}
	copy(destination[1:], destination[:4])
	if destination[0] != 1 || destination[1] != 1 || destination[4] != 4 {
		return "slices/overlap"
	}

	limited := values[1:3:3]
	limited = append(limited, 99)
	if len(limited) != 3 || limited[2] != 99 || values[3] != 4 {
		return "slices/full-expression"
	}

	var nilSlice []int
	if nilSlice != nil || len(nilSlice) != 0 || cap(nilSlice) != 0 {
		return "slices/nil"
	}
	return ""
}

func testStringsAndRunes() string {
	text := "Renvo"
	if len(text) != 5 || text[0] != 'R' || text[1:4] != "env" || text+"!" != "Renvo!" {
		return "strings/basic"
	}

	unicodeText := "A¢界"
	indices := 0
	runeSum := int32(0)
	for index, value := range unicodeText {
		indices = indices*10 + index
		runeSum += value
	}
	if indices != 13 || runeSum != 'A'+'¢'+'界' {
		return "strings/range"
	}

	bytes := []byte("hello")
	bytes[0] = 'H'
	if string(bytes) != "Hello" {
		return "strings/byte-conversion"
	}
	runes := []rune("Go界")
	if len(runes) != 3 || runes[2] != '界' || string(runes) != "Go界" {
		return "strings/rune-conversion"
	}
	return ""
}

func testStructsPointersAndMethods() string {
	p := point{x: 10, y: 20}
	if p.sum() != 30 {
		return "structs/method"
	}
	p.move(3, -4)
	if p != (point{13, 16}) || p.sum() != 29 {
		return "pointers/receiver"
	}

	alias := &p
	alias.x = 21
	if p.x != 21 || alias.y != 16 {
		return "pointers/alias"
	}

	created := new(pair)
	created.left, created.right = 19, 23
	if *created != (pair{left: 19, right: 23}) {
		return "pointers/new"
	}

	embedded := embeddedNumber{baseNumber: baseNumber{value: 42}, label: "answer"}
	if embedded.number() != 42 || embedded.label != "answer" {
		return "structs/embedding"
	}

	return ""
}

func apply(value int, operation func(int) int) int {
	return operation(value)
}

func testClosuresAndFunctionValues() string {
	base := 10
	add := func(value int) int {
		base += value
		return base
	}
	if add(2) != 12 || add(3) != 15 || base != 15 {
		return "closures/capture"
	}

	double := func(value int) int { return value * 2 }
	if apply(21, double) != 42 {
		return "functions/value"
	}

	var absent func()
	if absent != nil {
		return "functions/nil"
	}
	return ""
}

func interfaceNumber(value numbered) int {
	return value.number()
}

func classify(value interface{}) int {
	switch typed := value.(type) {
	case nil:
		return 0
	case int:
		return typed
	case string:
		return len(typed)
	case numbered:
		return typed.number()
	default:
		return -1
	}
}

func testInterfaces() string {
	var value named = animal{kind: "wren"}
	if value.name() != "wren" {
		return "interfaces/dispatch"
	}

	var empty interface{} = baseNumber{value: 42}
	number, ok := empty.(numbered)
	if !ok || number.number() != 42 || interfaceNumber(number) != 42 {
		return "interfaces/assertion"
	}
	text, textOK := empty.(string)
	if textOK || text != "" {
		return "interfaces/failed-assertion"
	}

	if classify(nil) != 0 || classify(7) != 7 || classify("four") != 4 || classify(empty) != 42 || classify(true) != -1 {
		return "interfaces/type-switch"
	}
	return ""
}

func testMaps() string {
	values := map[string]int{"one": 1, "two": 2}
	values["answer"] = 42
	value, ok := values["answer"]
	missing, missingOK := values["missing"]
	if !ok || value != 42 || missingOK || missing != 0 || len(values) != 3 {
		return "maps/access"
	}

	delete(values, "one")
	if _, ok := values["one"]; ok || len(values) != 2 {
		return "maps/delete"
	}

	total := 0
	for key, value := range values {
		total += len(key) + value
	}
	if total != 53 {
		return "maps/range"
	}
	var nilMap map[string]int
	if nilMap != nil || len(nilMap) != 0 {
		return "maps/nil"
	}
	return ""
}

var deferTrace int

func deferredOrder() int {
	defer func() { deferTrace = deferTrace*10 + 1 }()
	defer func() { deferTrace = deferTrace*10 + 2 }()
	defer func() { deferTrace = deferTrace*10 + 3 }()
	return 42
}

func deferredArgument() (result int) {
	value := 7
	defer func(captured int) { result += captured }(value)
	value = 99
	result = 35
	return
}

func recoveredValue() (result int) {
	defer func() {
		value := recover()
		if value == "expected panic" {
			result = 42
		}
	}()
	panic("expected panic")
}

func testDeferPanicRecover() string {
	deferTrace = 0
	if deferredOrder() != 42 || deferTrace != 321 {
		return "defer/order"
	}
	if deferredArgument() != 42 {
		return "defer/argument-capture"
	}
	if recoveredValue() != 42 {
		return "panic/recover"
	}
	return ""
}

func testComplexNumbers() string {
	left := complex(3.0, 4.0)
	right := complex(2.0, -1.0)
	sum := left + right
	if real(sum) != 5 || imag(sum) != 3 {
		return "complex/addition"
	}
	var small complex64 = complex(1.5, 2.5)
	if real(small) != 1.5 || imag(small) != 2.5 {
		return "complex/complex64"
	}
	return ""
}

func testUnsafeIntrinsics() string {
	var value layout
	size := unsafe.Sizeof(value)
	if unsafe.Sizeof(value.first) != 1 || size < 9 {
		return "unsafe/layout"
	}

	original := pair{left: 19, right: 23}
	raw := unsafe.Pointer(&original)
	converted := (*pair)(raw)
	if converted.left != 19 || converted.right != 23 {
		return "unsafe/pointer-conversion"
	}

	return ""
}

func testBuiltins() string {
	if builtinMinimum != 3 || builtinMaximum != 8 {
		return "builtins/min-max-integer"
	}
	if builtinStringMinimum != "apple" || builtinStringMaximum != "zoo" {
		return "builtins/min-max-string"
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
		return "builtins/min-max-evaluation"
	}
	values := []pair{{left: 1, right: 2}, {left: 3, right: 4}}
	clear(values)
	if len(values) != 2 || values[0] != (pair{}) || values[1] != (pair{}) {
		return "builtins/clear-slice"
	}
	return ""
}

func environmentValue(items []string, key string) string {
	for _, item := range items {
		if len(item) > len(key) && item[:len(key)] == key && item[len(key)] == '=' {
			return item[len(key)+1:]
		}
	}
	return ""
}

func testProcessAndFiles() string {
	if len(os.Args) != 2 || os.Args[1] != "suite-argument" {
		return "runtime/arguments"
	}
	if environmentValue(os.Environ(), "RENVO_SINGLE_FILE_MARKER") != "present" {
		return "runtime/environment"
	}

	const path = "single-file-suite.tmp"
	if err := os.WriteFile(path, []byte("renvo"), 0600); err != nil {
		return "files/write"
	}
	file, err := os.Open(path)
	if err != nil {
		return "files/open"
	}
	buffer := make([]byte, 8)
	count, readErr := file.Read(buffer)
	if count != 5 || readErr != nil || string(buffer[:count]) != "renvo" {
		file.Close()
		return "files/read"
	}
	count, readErr = file.Read(buffer)
	if count != 0 || readErr == nil {
		file.Close()
		return "files/eof"
	}
	if err := file.Close(); err != nil {
		return "files/close"
	}
	if err := file.Close(); err == nil {
		return "files/double-close"
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		return "files/readdir"
	}
	found := false
	for _, entry := range entries {
		if entry.Name() == path && !entry.IsDir() {
			found = true
		}
	}
	if !found {
		return "files/readdir-entry"
	}
	return ""
}

func main() {
	name := testInitialization()
	if name == "" {
		name = testArithmetic()
	}
	if name == "" {
		name = testIntegerWidthsAndConversions()
	}
	if name == "" {
		name = testComparisonAndBoolean()
	}
	if name == "" {
		name = testBitwiseAndShifts()
	}
	if name == "" {
		name = testControlFlow()
	}
	if name == "" {
		name = testFunctions()
	}
	if name == "" {
		name = testArrays()
	}
	if name == "" {
		name = testSlices()
	}
	if name == "" {
		name = testStringsAndRunes()
	}
	if name == "" {
		name = testStructsPointersAndMethods()
	}
	if name == "" {
		name = testClosuresAndFunctionValues()
	}
	if name == "" {
		name = testInterfaces()
	}
	if name == "" {
		name = testMaps()
	}
	if name == "" {
		name = testDeferPanicRecover()
	}
	if name == "" {
		name = testComplexNumbers()
	}
	if name == "" {
		name = testUnsafeIntrinsics()
	}
	if name == "" {
		name = testBuiltins()
	}
	if name == "" {
		name = testProcessAndFiles()
	}
	if name != "" {
		failure(name)
		return
	}
	print("PASS\n")
}
