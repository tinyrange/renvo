//go:build !renvo

// Package difftest generates valid Go programs for differential compiler
// testing and reduces programs which behave differently under two compilers.
package difftest

import (
	"fmt"
	"go/format"
	"strings"
)

// Generate returns a deterministic standalone Go program containing count
// independently removable language-feature cases.
func Generate(seed uint64, count int) ([]byte, error) {
	return generate(seed, count, caseFamilies)
}

// GenerateFamily restricts generation to one named feature family.
func GenerateFamily(seed uint64, count int, name string) ([]byte, error) {
	for index, candidate := range caseFamilyNames {
		if candidate == name {
			return generate(seed, count, caseFamilies[index:index+1])
		}
	}
	return nil, fmt.Errorf("unknown feature family %q", name)
}

// Families returns the stable names accepted by GenerateFamily.
func Families() []string {
	return append([]string(nil), caseFamilyNames...)
}

func generate(seed uint64, count int, available []func(int, *randomSource) string) ([]byte, error) {
	if count < 1 {
		count = 1
	}
	random := newRandom(seed)
	var declarations strings.Builder
	var calls strings.Builder
	families := make([]int, len(available))
	for i := 0; i < count; i++ {
		at := i % len(families)
		if at == 0 {
			for family := range families {
				families[family] = family
			}
			for family := len(families) - 1; family > 0; family-- {
				other := int(random.next() % uint64(family+1))
				families[family], families[other] = families[other], families[family]
			}
		}
		family := families[at]
		declaration := available[family](i, random)
		declarations.WriteString(declaration)
		fmt.Fprintf(&calls, "\th = (h ^ uint64(case%d())) * 1099511628211\n", i)
	}

	source := fmt.Sprintf(`package main

%s
func emitHash(value uint64) {
	const digits = "0123456789abcdef"
	var output [16]byte
	for index := 15; index >= 0; index-- {
		output[index] = digits[value&15]
		value >>= 4
	}
	print(string(output[:]))
	print("\n")
}

func main() {
	h := uint64(1469598103934665603)
%s	emitHash(h)
}
`, declarations.String(), calls.String())
	formatted, err := format.Source([]byte(source))
	if err != nil {
		return nil, fmt.Errorf("format generated seed %d: %w", seed, err)
	}
	return formatted, nil
}

type randomSource struct {
	state uint64
}

func newRandom(seed uint64) *randomSource {
	return &randomSource{state: seed + 0x9e3779b97f4a7c15}
}

func (r *randomSource) next() uint64 {
	r.state += 0x9e3779b97f4a7c15
	z := r.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

func (r *randomSource) small() int64 {
	return int64(r.next()%101) - 50
}

func (r *randomSource) positive() uint64 {
	return r.next()%63 + 1
}

var caseFamilies = []func(int, *randomSource) string{
	arithmeticCase,
	expressionTreeCase,
	controlCase,
	arraySliceCase,
	structMethodCase,
	interfaceCase,
	mapCase,
	stringCase,
	closureCase,
	deferCase,
	complexCase,
	multipleResultCase,
	pointerCase,
	embeddedMethodCase,
	recoverCase,
	shortCircuitCase,
	conversionCase,
	comparableCase,
	arrayComparableCase,
	typeSwitchCase,
	variadicCase,
	rangeCase,
	returnedClosureCase,
	deferredPairCase,
	fallthroughCase,
	builtinCase,
	recursiveCase,
	assignmentOrderCase,
	typedNilInterfaceCase,
	floatCase,
	boolMapCase,
	mapAssignmentOrderCase,
	sliceAliasingCase,
	deferArgumentTimingCase,
	methodExpressionCase,
	valueMethodExpressionCase,
	namedTypeCase,
	labeledLoopCase,
	expressionlessSwitchCase,
	bareReturnCase,
	argumentOrderCase,
	nilContainerCase,
	arrayPointerCase,
	shadowingCase,
	switchShadowingCase,
	forShadowingCase,
	methodValueTimingCase,
	calleeOrderCase,
	nestedArgumentOrderCase,
	compoundIndexOrderCase,
	rangeCopyCase,
	unicodeRangeCase,
	interfaceMethodValueCase,
	namedContainerCase,
	randomStatementCase,
	randomSliceProgramCase,
	immediateFunctionCallCase,
}

var caseFamilyNames = []string{
	"arithmetic",
	"expression-tree",
	"control-flow",
	"array-slice",
	"struct-method",
	"interface",
	"scalar-map",
	"string",
	"closure",
	"defer",
	"complex",
	"multiple-results",
	"pointer",
	"embedded-method",
	"recover",
	"short-circuit",
	"conversion",
	"comparable-struct",
	"comparable-array",
	"type-switch",
	"variadic",
	"range",
	"returned-closure",
	"deferred-results",
	"fallthrough",
	"builtins",
	"recursion",
	"assignment-order",
	"typed-nil-interface",
	"float",
	"bool-map",
	"map-assignment-order",
	"slice-aliasing",
	"defer-argument-timing",
	"pointer-method-expression",
	"value-method-expression",
	"named-types",
	"labeled-loop",
	"expressionless-switch",
	"bare-return",
	"argument-order",
	"nil-containers",
	"pointer-array",
	"if-shadowing",
	"switch-shadowing",
	"for-shadowing",
	"method-value-timing",
	"callee-order",
	"nested-argument-order",
	"compound-index-order",
	"range-copy",
	"unicode-range",
	"interface-method-value",
	"named-containers",
	"random-statements",
	"random-slice-program",
	"immediate-function-call",
}

func expressionTreeCase(index int, random *randomSource) string {
	a, b, c, d := random.next(), random.next(), random.next(), random.next()
	expression := randomUintExpression(random, 4)
	return fmt.Sprintf(`
func case%d() int64 {
	a := uint64(%d)
	b := uint64(%d)
	c := uint64(%d)
	d := uint64(%d)
	value := %s
	value ^= a ^ b ^ c ^ d
	value ^= uint64(int64(int16(value)))
	return int64(value)
}
`, index, a, b, c, d, expression)
}

func randomUintExpression(random *randomSource, depth int) string {
	if depth == 0 || random.next()%5 == 0 {
		switch random.next() % 4 {
		case 0:
			return "a"
		case 1:
			return "b"
		case 2:
			return "c"
		default:
			return "d"
		}
	}
	left := randomUintExpression(random, depth-1)
	if random.next()%6 == 0 {
		return fmt.Sprintf("((%s) << %d)", left, random.next()%64)
	}
	right := randomUintExpression(random, depth-1)
	operators := [...]string{"+", "-", "*", "^", "|", "&", "&^"}
	return fmt.Sprintf("((%s) %s (%s))", left, operators[random.next()%uint64(len(operators))], right)
}

func arithmeticCase(index int, random *randomSource) string {
	a, b := random.next(), random.positive()
	left, right := random.next()%31, random.next()%31
	return fmt.Sprintf(`
func case%d() int64 {
	a := uint32(%d)
	b := uint32(%d)
	c := ((a << %d) | (a >> %d)) ^ (b * 33)
	c += a/b + a%%b
	return int64(int32(c))
}
`, index, uint32(a), uint32(b), left, right)
}

func controlCase(index int, random *randomSource) string {
	limit := random.next()%17 + 4
	divisor := random.next()%5 + 2
	return fmt.Sprintf(`
func case%d() int64 {
	total := int64(0)
	for i := 0; i < %d; i++ {
		if i%%%d == 0 { continue }
		switch i & 3 {
		case 0: total += int64(i * i)
		case 1: total -= int64(i + %d)
		default: total ^= int64(i * %d)
		}
		if total > 10000 { break }
	}
	return total
}
`, index, limit, divisor, random.next()%29, random.next()%13+1)
}

func arraySliceCase(index int, random *randomSource) string {
	a, b, c, d := random.small(), random.small(), random.small(), random.small()
	return fmt.Sprintf(`
func case%d() int64 {
	values := [...]int64{%d, %d, %d, %d}
	window := values[1:3:4]
	window = append(window, values[0])
	copy(values[:2], window[1:])
	return values[0]*3 + values[1]*5 + window[2]*7 + int64(len(window)+cap(window))
}
`, index, a, b, c, d)
}

func structMethodCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type pair%d struct { left int32; right uint16 }
func (p pair%d) mix(scale int32) int64 { return int64(p.left*scale) - int64(p.right) }
func case%d() int64 {
	p := pair%d{left: %d, right: %d}
	method := p.mix
	p.left += %d
	return method(%d) + p.mix(%d)
}
`, index, index, index, index, int32(random.small()), uint16(random.next()), int32(random.small()), int32(random.small()), int32(random.small()))
}

func interfaceCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type valueInterface%d interface { value(int64) int64 }
type valueImpl%d int64
func (v valueImpl%d) value(x int64) int64 { return int64(v)*x + %d }
func case%d() int64 {
	var dynamic valueInterface%d = valueImpl%d(%d)
	first := dynamic.value(%d)
	concrete, ok := dynamic.(valueImpl%d)
	if !ok { return -1 }
	return first + int64(concrete)
}
`, index, index, index, random.small(), index, index, index, random.small(), random.small(), index)
}

func mapCase(index int, random *randomSource) string {
	k1, k2 := random.next()%31+1, random.next()%31+40
	v1, v2 := random.small(), random.small()
	return fmt.Sprintf(`
func case%d() int64 {
	values := map[int]int64{%d: %d, %d: %d}
	values[%d] += values[%d]
	missing, present := values[-1]
	delete(values, %d)
	if present { return missing - 1000 }
	return values[%d] + int64(len(values))*17
}
`, index, k1, v1, k2, v2, k1, k2, k2, k1)
}

func stringCase(index int, random *randomSource) string {
	letter := byte('a' + random.next()%26)
	return fmt.Sprintf(`
func case%d() int64 {
	text := "renvo-" + "differential"
	bytes := []byte(text[2:11])
	bytes[3] = %d
	changed := string(bytes)
	result := int64(len(changed))*31 + int64(changed[0]) + int64(changed[len(changed)-1])
	if changed == text { result = -result }
	return result
}
`, index, letter)
}

func closureCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	base := int64(%d)
	step := int64(%d)
	apply := func(value int64) int64 { base += step; return value*base + step }
	first := apply(%d)
	second := apply(first & 15)
	return first ^ second ^ base
}
`, index, random.small(), random.small(), random.small())
}

func deferCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func deferred%d(value int64) (result int64) {
	result = value
	defer func(delta int64) { result = result*3 + delta }(%d)
	defer func() { result ^= %d }()
	return result + %d
}
func case%d() int64 { return deferred%d(%d) }
`, index, random.small(), random.next()%127, random.small(), index, index, random.small())
}

func complexCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	a := complex(float64(%d), float64(%d))
	b := complex(float64(%d), float64(%d))
	c := a*b + complex(real(a), -imag(b))
	return int64(real(c))*17 + int64(imag(c))*29
}
`, index, random.small(), random.small(), random.small(), random.small())
}

func multipleResultCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func results%d(value int64) (int64, int64) { return value + %d, value ^ %d }
func case%d() int64 {
	a, b := results%d(%d)
	a, b = b, a
	return a*7 + b*11
}
`, index, random.small(), random.next()%127, index, index, random.small())
}

func pointerCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	values := [3]int64{%d, %d, %d}
	pointer := &values[1]
	*pointer += values[0]
	other := new(int64)
	*other = values[2] - *pointer
	return *pointer*13 + *other*19
}
`, index, random.small(), random.small(), random.small())
}

func embeddedMethodCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type inner%d struct { value int64 }
func (v *inner%d) bump(delta int64) int64 { v.value += delta; return v.value }
type outer%d struct { inner%d; extra int64 }
func case%d() int64 {
	v := outer%d{inner%d: inner%d{value: %d}, extra: %d}
	method := v.bump
	return method(%d)*5 + v.value + v.extra
}
`, index, index, index, index, index, index, index, index, random.small(), random.small(), random.small())
}

func recoverCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func recovered%d(value int) (result int64) {
	defer func() {
		if recovered := recover(); recovered != nil { result += int64(recovered.(int)) }
	}()
	result = int64(value * 3)
	panic(value + %d)
}
func case%d() int64 { return recovered%d(%d) }
`, index, int(random.next()%31), index, index, int(random.next()%31))
}

func shortCircuitCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	state := int64(%d)
	touch := func(value int64) bool { state = state*5 + value; return value&1 == 0 }
	left := touch(%d) && touch(%d)
	right := touch(%d) || touch(%d)
	if left != right { state = -state }
	return state
}
`, index, random.small(), random.small(), random.small(), random.small(), random.small())
}

func conversionCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	signed := int8(%d)
	unsigned := uint16(%d)
	widened := uint64(uint8(signed))<<24 | uint64(unsigned)
	narrowed := int32(uint32(widened ^ uint64(%d)))
	return int64(narrowed) + int64(int16(unsigned)) - int64(signed)
}
`, index, int8(random.next()), uint16(random.next()), uint32(random.next()))
}

func comparableCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type comparable%d struct { number int16; bytes [2]byte; text string }
func case%d() int64 {
	left := comparable%d{number: %d, bytes: [2]byte{%d, %d}, text: "same"}
	right := left
	result := int64(0)
	if left == right { result += 17 }
	right.bytes[1] ^= %d
	if left != right { result += 31 }
	return result + int64(right.number)
}
`, index, index, index, int16(random.next()), byte(random.next()), byte(random.next()), byte(random.next()%255+1))
}

func arrayComparableCase(index int, random *randomSource) string {
	left, right := random.small(), random.small()
	return fmt.Sprintf(`
func case%d() int64 {
	value := int64(0)
	if [2]int64{%d, %d} == [2]int64{%d, %d} { value += 47 }
	if [2]int64{%d, %d} != [2]int64{%d, %d} { value += 83 }
	return value
}
`, index, left, right, left, right, left, right, right, left)
}

func typeSwitchCase(index int, random *randomSource) string {
	choice := random.next() % 3
	value := fmt.Sprintf("int64(%d)", random.small())
	if choice == 1 {
		value = `"switch-value"`
	} else if choice == 2 {
		value = fmt.Sprintf("switchValue%d{number: %d}", index, random.small())
	}
	return fmt.Sprintf(`
type switchValue%d struct { number int64 }
func case%d() int64 {
	var dynamic interface{} = %s
	switch value := dynamic.(type) {
	case int64: return value*3 + %d
	case string: return int64(len(value))*5
	case switchValue%d: return value.number*7
	default: return -1
	}
}
`, index, index, value, random.small(), index)
}

func variadicCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func variadic%d(base int64, values ...int64) int64 {
	total := base
	for index, value := range values { total += int64(index+1)*value }
	return total
}
func case%d() int64 {
	values := []int64{%d, %d, %d}
	return variadic%d(%d, values...) + variadic%d(%d, %d, %d)
}
`, index, index, random.small(), random.small(), random.small(), index, random.small(), index, random.small(), random.small(), random.small())
}

func rangeCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	values := [...]int16{%d, %d, %d, %d}
	total := int64(0)
	for index, value := range values { total += int64(index+1) * int64(value) }
	for index, value := range "A¢日" { total += int64(index)*3 + int64(value) }
	return total
}
`, index, int16(random.next()), int16(random.next()), int16(random.next()), int16(random.next()))
}

func returnedClosureCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func closureFactory%d(base int64) func(int64) int64 {
	state := base
	return func(value int64) int64 { state = state*3 + value; return state }
}
func case%d() int64 {
	closure := closureFactory%d(%d)
	return closure(%d)*7 + closure(%d)
}
`, index, index, index, random.small(), random.small(), random.small())
}

func deferredPairCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func deferredPair%d(value int64) (left int64, right int64) {
	defer func() { left, right = right + %d, left - %d }()
	return value, value + %d
}
func case%d() int64 {
	left, right := deferredPair%d(%d)
	return left*11 + right*13
}
`, index, random.small(), random.small(), random.small(), index, index, random.small())
}

func fallthroughCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	value := uint64(%d)
	result := int64(0)
	switch value & 3 {
	case 0: result += 3; fallthrough
	case 1: result = result*5 + 7; fallthrough
	case 2: result = result*11 - 2
	default: result = 19
	}
	return result
}
`, index, random.next())
}

func builtinCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	values := []int64{%d, %d, %d, %d}
	clear(values[1:3])
	mapping := map[string]int{"first": 2, "second": 4}
	clear(mapping)
	return max(values[0], values[3])*7 + min(values[0], values[3])*11 + int64(len(mapping))
}
`, index, random.small(), random.small(), random.small(), random.small())
}

func recursiveCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func recursive%d(value int64, depth int) int64 {
	if depth == 0 { return value }
	if depth&1 == 0 { return recursive%d(value*3+int64(depth), depth-1) }
	return recursive%d(value^int64(depth*7), depth-1)
}
func case%d() int64 { return recursive%d(%d, %d) }
`, index, index, index, index, index, random.small(), random.next()%7+2)
}

func assignmentOrderCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	values := []int64{%d, %d, %d}
	position := 0
	position, values[position] = 1, %d
	values[position], position = int64(position)+values[0], 2
	return values[0]*3 + values[1]*5 + values[2]*7 + int64(position)
}
`, index, random.small(), random.small(), random.small(), random.small())
}

func typedNilInterfaceCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	var pointer *int64
	var dynamic interface{} = pointer
	result := int64(%d)
	if dynamic == nil { result += 1000 }
	asserted, ok := dynamic.(*int64)
	if ok && asserted == nil { result += 37 }
	var empty interface{}
	if empty == nil { result += 53 }
	return result
}
`, index, random.small())
}

func floatCase(index int, random *randomSource) string {
	a := int64(random.next()%81) - 40
	b := int64(random.next()%15) + 1
	c := int64(random.next()%81) - 40
	return fmt.Sprintf(`
func case%d() int64 {
	a := float64(%d) / 4.0
	b := float64(%d)
	c := float64(%d) / 4.0
	value := a*b + c
	return int64(value*4.0) + int64(a-c)
}
`, index, a, b, c)
}

func boolMapCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	values := map[bool]int64{true: %d, false: %d}
	values[true] ^= values[false]
	delete(values, false)
	return values[true] + int64(len(values))*23
}
`, index, random.small(), random.small())
}

func mapAssignmentOrderCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	values := map[string]int64{"zero": %d, "one": %d, "two": %d}
	key := "zero"
	key, values[key] = "one", %d
	first, present := values["zero"]
	missing, absent := values["missing"]
	if !present || absent { return -1000 }
	return int64(len(key))*13 + first*17 + missing
}
`, index, random.small(), random.small(), random.small(), random.small())
}

func sliceAliasingCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	array := [...]int64{%d, %d, %d, %d, %d}
	window := array[1:4:5]
	alias := window[1:]
	alias[0] ^= %d
	window = append(window, %d)
	copy(window[1:], window[:2])
	return array[0]*3 + array[1]*5 + array[2]*7 + array[3]*11 + array[4]*13
}
`, index, random.small(), random.small(), random.small(), random.small(), random.small(), random.small(), random.small())
}

func deferArgumentTimingCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func deferTiming%d(value int64) (result int64) {
	delta := int64(%d)
	defer func(captured int64) { result = result*5 + captured }(delta)
	defer func() { result = result*7 + delta }()
	delta = %d
	return value + delta
}
func case%d() int64 { return deferTiming%d(%d) }
`, index, random.small(), random.small(), index, index, random.small())
}

func methodExpressionCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type meter%d int64
func (value *meter%d) add(delta int64) { *value += meter%d(delta) }
func case%d() int64 {
	value := meter%d(%d)
	add := (*meter%d).add
	add(&value, %d)
	return int64(value)
}
`, index, index, index, index, index, random.small(), index, random.small())
}

func valueMethodExpressionCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type scale%d int64
func (value scale%d) apply(factor int64) int64 { return int64(value)*factor }
func case%d() int64 {
	value := scale%d(%d)
	apply := scale%d.apply
	return apply(value, %d)
}
`, index, index, index, index, random.small(), index, random.small())
}

func namedTypeCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type signed%d int16
type unsigned%d uint8
type label%d string
func case%d() int64 {
	left := signed%d(%d)
	right := unsigned%d(%d)
	left += signed%d(right)
	text := label%d("renvo")
	if text == label%d("renvo") { left ^= signed%d(len(text)) }
	return int64(left)<<2 + int64(right>>1)
}
`, index, index, index, index, index, int16(random.next()), index, uint8(random.next()), index, index, index, index)
}

func labeledLoopCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	total := int64(%d)
outer:
	for row := 0; row < %d; row++ {
		for column := 0; column < %d; column++ {
			if row+column == %d { continue outer }
			if row*column > %d { break outer }
			total = total*3 + int64(row-column)
		}
	}
	return total
}
`, index, random.small(), random.next()%5+2, random.next()%5+2, random.next()%7, random.next()%9+2)
}

func expressionlessSwitchCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	switch value := int64(%d); {
	case value < -10:
		return value * 3
	case value == 0:
		return 17
	case value < 10:
		return value*5 + 1
	default:
		return value ^ 31
	}
}
`, index, random.small())
}

func bareReturnCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func bareReturn%d(value int64) (result int64) {
	result = value*3 + %d
	defer func() { result = result*5 + %d }()
	if value&1 == 0 { result ^= %d; return }
	result += %d
	return
}
func case%d() int64 { return bareReturn%d(%d) }
`, index, random.small(), random.small(), random.small(), random.small(), index, index, random.small())
}

func argumentOrderCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func combine%d(left, middle, right int64) int64 { return left*100 + middle*10 + right }
func case%d() int64 {
	state := int64(%d)
	next := func() int64 { state = state*3 + %d; return state }
	return combine%d(next(), next(), next()) + state
}
`, index, index, random.small(), random.small(), index)
}

func nilContainerCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	var values []int64
	values = append(values, %d, %d)
	destination := make([]int64, 3)
	copied := copy(destination[1:], values)
	var mapping map[string]int64
	zero, present := mapping["missing"]
	delete(mapping, "missing")
	if values == nil || mapping != nil || present { return -1000 }
	return destination[1]*3 + destination[2]*5 + zero + int64(copied+len(mapping))
}
`, index, random.small(), random.small())
}

func arrayPointerCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	array := &[4]int64{%d, %d, %d, %d}
	window := array[1:3]
	window[0] += %d
	return (*array)[0]*3 + array[1]*5 + array[2]*7 + int64(len(array)+cap(window))
}
`, index, random.small(), random.small(), random.small(), random.small(), random.small())
}

func shadowingCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	value := int64(%d)
	result := value
	if value := value + %d; value > 0 { result += value } else { result -= value }
	return result + value
}
`, index, random.small(), random.small())
}

func switchShadowingCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	value := int64(%d)
	result := value
	switch value := value ^ %d; value & 1 {
	case 0: result = result*3 + value
	default: result = result*5 - value
	}
	return result + value
}
`, index, random.small(), random.small())
}

func forShadowingCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	value := int64(%d)
	result := int64(0)
	for value := value + %d; value != 1000; value++ {
		result = value
		break
	}
	return result*3 + value
}
`, index, random.small(), random.small())
}

func methodValueTimingCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type snapshot%d int64
func (value snapshot%d) read(delta int64) int64 { return int64(value) + delta }
func case%d() int64 {
	value := snapshot%d(%d)
	read := value.read
	value += snapshot%d(%d)
	return read(%d)*17 + int64(value)
}
`, index, index, index, index, random.small(), index, random.small(), random.small())
}

func calleeOrderCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	state := int64(%d)
	choose := func() func(int64) int64 {
		state = state*3 + %d
		return func(value int64) int64 { return state*11 + value }
	}
	argument := func() int64 { state = state*5 + %d; return state }
	return choose()(argument()) + state
}
`, index, random.small(), random.small(), random.small())
}

func nestedArgumentOrderCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func nestedCombine%d(left, right int64) int64 { return left*100 + right }
func case%d() int64 {
	state := int64(%d)
	next := func() int64 { state = state*3 + %d; return state }
	return nestedCombine%d(next()+1, next()*2) + state
}
`, index, index, random.small(), random.small(), index)
}

func compoundIndexOrderCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	values := []int64{%d, %d}
	state := int64(%d)
	position := func() int { state = state*3 + 1; return 0 }
	delta := func() int64 { state = state*5 + 2; return state }
	values[position()] += delta()
	return values[0]*7 + values[1]*11 + state
}
`, index, random.small(), random.small(), random.small())
}

func rangeCopyCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	values := [3]int64{%d, %d, %d}
	total := int64(0)
	for position, value := range values {
		value += int64(position + 1)
		total = total*7 + value
	}
	return total + values[0]*3 + values[1]*5 + values[2]*11
}
`, index, random.small(), random.small(), random.small())
}

func unicodeRangeCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	text := "A¢€𐍈"
	result := int64(%d)
	for offset, value := range text {
		result = result*7 + int64(offset)*3 + int64(value)
	}
	return result + int64(len(text))
}
`, index, random.small())
}

func interfaceMethodValueCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type operation%d interface { apply(int64) int64 }
type multiplier%d int64
func (value multiplier%d) apply(input int64) int64 { return int64(value)*input }
func case%d() int64 {
	var operation operation%d = multiplier%d(%d)
	apply := operation.apply
	return apply(%d)
}
`, index, index, index, index, index, index, random.small(), random.small())
}

func namedContainerCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type numbers%d []int64
type lookup%d map[string]int64
func case%d() int64 {
	values := numbers%d{%d, %d}
	values = append(values, %d)
	mapping := lookup%d{"left": values[0], "right": values[2]}
	delete(mapping, "left")
	return values[0]*3 + values[1]*5 + values[2]*7 + mapping["right"] + int64(len(mapping))
}
`, index, index, index, index, random.small(), random.small(), random.small(), index)
}

func randomStatementCase(index int, random *randomSource) string {
	var body strings.Builder
	for step := 0; step < 8; step++ {
		value := random.small()
		switch random.next() % 8 {
		case 0:
			fmt.Fprintf(&body, "\ta = (a ^ int64(%d)) + b\n", value)
		case 1:
			fmt.Fprintf(&body, "\tb = b*3 - a + int64(%d)\n", value)
		case 2:
			fmt.Fprintf(&body, "\tif probe := a + int64(%d); probe&1 == 0 { total += probe } else { total -= probe }\n", value)
		case 3:
			fmt.Fprintf(&body, "\tfor loop%d := 0; loop%d < %d; loop%d++ { total += a ^ int64(loop%d); a += int64(loop%d) }\n", step, step, random.next()%3+1, step, step, step)
		case 4:
			fmt.Fprintf(&body, "\tswitch (a + b + int64(%d)) & 3 { case 0: total += a; case 1: total -= b; default: total ^= a + b }\n", value)
		case 5:
			fmt.Fprintf(&body, "\tapply%d := func(delta int64) { b += delta; total = total*5 + b }; apply%d(int64(%d))\n", step, step, value)
		case 6:
			fmt.Fprintf(&body, "\ta, b = b + int64(%d), a - int64(%d)\n", value, value)
		case 7:
			fmt.Fprintf(&body, "\ttotal += randomHelper%d(a+int64(%d), b-int64(%d))\n", index, value, value)
		}
	}
	return fmt.Sprintf(`
func randomHelper%d(left, right int64) int64 { return left*7 + right*11 }
func case%d() int64 {
	a := int64(%d)
	b := int64(%d)
	total := int64(%d)
%s	return total*13 + a*17 + b*19
}
`, index, index, random.small(), random.small(), random.small(), body.String())
}

func randomSliceProgramCase(index int, random *randomSource) string {
	var body strings.Builder
	for step := 0; step < 8; step++ {
		first := int(random.next() % 5)
		second := int(random.next() % 5)
		value := random.small()
		switch random.next() % 5 {
		case 0:
			fmt.Fprintf(&body, "\tvalues[%d] += int64(%d)\n", first, value)
		case 1:
			fmt.Fprintf(&body, "\tcopy(values[%d:%d], values[%d:%d])\n", first, first+1, second, second+1)
		case 2:
			fmt.Fprintf(&body, "\twindow%d := values[%d:%d]; window%d[0] ^= int64(%d)\n", step, first, first+1, step, value)
		case 3:
			fmt.Fprintf(&body, "\tvalues = append(values, int64(%d)); values = values[1:]\n", value)
		case 4:
			fmt.Fprintf(&body, "\tvalues[%d], values[%d] = values[%d], values[%d]\n", first, second, second, first)
		}
	}
	return fmt.Sprintf(`
func case%d() int64 {
	values := []int64{%d, %d, %d, %d, %d}
%s	result := int64(0)
	for position, value := range values { result = result*7 + int64(position)*3 + value }
	return result + int64(len(values))*11 + int64(cap(values))*13
}
`, index, random.small(), random.small(), random.small(), random.small(), random.small(), body.String())
}

func immediateFunctionCallCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	value := int64(%d)
	func(delta int64) { value = value*3 + delta }(int64(%d))
	return value
}
`, index, random.small(), random.small())
}
