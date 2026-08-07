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
	return generate(seed, count, caseFamilies, defaultGenerationPolicy)
}

// GenerateSwarm chooses a deterministic subset of language-feature families
// for each seed. Varying both the selected features and their density avoids
// making every large generated program exercise the same maximal feature set.
func GenerateSwarm(seed uint64, count int) ([]byte, error) {
	random := newRandom(seed ^ 0x535741524d)
	density := random.next()%3 + 1
	selected := make([]func(int, *randomSource) string, 0, len(caseFamilies))
	for _, family := range caseFamilies {
		if random.next()%4 < density {
			selected = append(selected, family)
		}
	}
	if len(selected) == 0 {
		selected = append(selected, caseFamilies[random.next()%uint64(len(caseFamilies))])
	}
	return generate(seed, count, selected, defaultGenerationPolicy)
}

// GeneratePolicy generates with one named policy that changes the shape and
// density of code while retaining the complete feature-family set.
func GeneratePolicy(seed uint64, count int, name string) ([]byte, error) {
	policy, ok := findGenerationPolicy(name)
	if !ok {
		return nil, fmt.Errorf("unknown generation policy %q", name)
	}
	return generate(seed, count, caseFamilies, policy)
}

// GenerateFamily restricts generation to one named feature family.
func GenerateFamily(seed uint64, count int, name string) ([]byte, error) {
	return generateFamily(seed, count, name, defaultGenerationPolicy)
}

// GenerateFamilyPolicy combines a focused feature family with a code-shape
// policy, making policy effects independently measurable and reproducible.
func GenerateFamilyPolicy(seed uint64, count int, familyName string, policyName string) ([]byte, error) {
	policy, ok := findGenerationPolicy(policyName)
	if !ok {
		return nil, fmt.Errorf("unknown generation policy %q", policyName)
	}
	return generateFamily(seed, count, familyName, policy)
}

func generateFamily(seed uint64, count int, name string, policy generationPolicy) ([]byte, error) {
	for index, candidate := range caseFamilyNames {
		if candidate == name {
			return generate(seed, count, caseFamilies[index:index+1], policy)
		}
	}
	return nil, fmt.Errorf("unknown feature family %q", name)
}

// Families returns the stable names accepted by GenerateFamily.
func Families() []string {
	return append([]string(nil), caseFamilyNames...)
}

// Policies returns the stable names accepted by GeneratePolicy.
func Policies() []string {
	names := make([]string, len(generationPolicies))
	for index, policy := range generationPolicies {
		names[index] = policy.name
	}
	return names
}

func generate(seed uint64, count int, available []func(int, *randomSource) string, policy generationPolicy) ([]byte, error) {
	if count < 1 {
		count = 1
	}
	random := newRandom(seed)
	random.policy = policy
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
	state  uint64
	policy generationPolicy
}

type generationPolicy struct {
	name            string
	expressionDepth int
	statementSteps  int
	sliceSteps      int
	dataFlowSteps   int
	aliasCount      int
	materialization int
}

var defaultGenerationPolicy = generationPolicy{
	expressionDepth: 4,
	statementSteps:  8,
	sliceSteps:      8,
	dataFlowSteps:   6,
	aliasCount:      2,
	materialization: -1,
}

var generationPolicies = []generationPolicy{
	{name: "shallow", expressionDepth: 2, statementSteps: 4, sliceSteps: 4, dataFlowSteps: 3, aliasCount: 1, materialization: 1},
	{name: "deep-expression", expressionDepth: 8, statementSteps: 8, sliceSteps: 8, dataFlowSteps: 6, aliasCount: 2, materialization: 0},
	{name: "long-dataflow", expressionDepth: 4, statementSteps: 12, sliceSteps: 8, dataFlowSteps: 20, aliasCount: 2, materialization: 2},
	{name: "alias-dense", expressionDepth: 4, statementSteps: 8, sliceSteps: 16, dataFlowSteps: 8, aliasCount: 8, materialization: 1},
	{name: "materialize-direct", expressionDepth: 4, statementSteps: 8, sliceSteps: 8, dataFlowSteps: 8, aliasCount: 2, materialization: 0},
	{name: "materialize-local", expressionDepth: 4, statementSteps: 8, sliceSteps: 8, dataFlowSteps: 8, aliasCount: 2, materialization: 1},
	{name: "materialize-call", expressionDepth: 4, statementSteps: 8, sliceSteps: 8, dataFlowSteps: 8, aliasCount: 2, materialization: 2},
	{name: "materialize-interface", expressionDepth: 4, statementSteps: 8, sliceSteps: 8, dataFlowSteps: 8, aliasCount: 2, materialization: 3},
	{name: "materialize-index", expressionDepth: 4, statementSteps: 8, sliceSteps: 8, dataFlowSteps: 8, aliasCount: 2, materialization: 4},
	{name: "materialize-pointer", expressionDepth: 4, statementSteps: 8, sliceSteps: 8, dataFlowSteps: 8, aliasCount: 2, materialization: 5},
	{name: "materialize-map", expressionDepth: 4, statementSteps: 8, sliceSteps: 8, dataFlowSteps: 8, aliasCount: 2, materialization: 6},
	{name: "stress", expressionDepth: 7, statementSteps: 16, sliceSteps: 16, dataFlowSteps: 20, aliasCount: 8, materialization: -1},
}

func findGenerationPolicy(name string) (generationPolicy, bool) {
	for _, policy := range generationPolicies {
		if policy.name == name {
			return policy, true
		}
	}
	return generationPolicy{}, false
}

func newRandom(seed uint64) *randomSource {
	return &randomSource{state: seed + 0x9e3779b97f4a7c15, policy: defaultGenerationPolicy}
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
	compositeMapCase,
	interfaceMapCase,
	mapRangeMutationCase,
	mapOperandOrderCase,
	floatMapCase,
	complexMapCase,
	pointerMapCase,
	nestedCompositeFlowCase,
	aggregateAssignmentCase,
	mapCompositeValueCase,
	interfaceComparisonCase,
	complex64FlowCase,
	appendOverlapCase,
	closureAggregateCase,
	deferPanicChainCase,
	interfaceRoundTripCase,
	stringRuneRoundTripCase,
	structPointerFlowCase,
	complexDivisionCase,
	switchInitializerVariantsCase,
	emiDeadCodeCase,
	metamorphicExpressionCase,
	policyDataFlowCase,
	policyAliasCase,
	policyMaterializationCase,
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
	"composite-map",
	"interface-map",
	"map-range-mutation",
	"map-operand-order",
	"float-map",
	"complex-map",
	"pointer-map",
	"nested-composite-flow",
	"aggregate-assignment",
	"map-composite-value",
	"interface-comparison",
	"complex64-flow",
	"append-overlap",
	"closure-aggregate",
	"defer-panic-chain",
	"interface-round-trip",
	"string-rune-round-trip",
	"struct-pointer-flow",
	"complex-division",
	"switch-initializer-variants",
	"emi-dead-code",
	"metamorphic-expression",
	"policy-data-flow",
	"policy-alias",
	"policy-materialization",
}

func expressionTreeCase(index int, random *randomSource) string {
	a, b, c, d := random.next(), random.next(), random.next(), random.next()
	expression := randomUintExpression(random, random.policy.expressionDepth)
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
	for step := 0; step < random.policy.statementSteps; step++ {
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
	for step := 0; step < random.policy.sliceSteps; step++ {
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

func compositeMapCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type compositeKey%d struct { left int32; right [2]int16 }
type compositeValue%d struct { total int64; slots [2]int32 }
func case%d() int64 {
	first := compositeKey%d{left: %d, right: [2]int16{%d, %d}}
	second := compositeKey%d{left: %d, right: [2]int16{%d, %d}}
	values := map[compositeKey%d]compositeValue%d{
		first: {total: %d, slots: [2]int32{%d, %d}},
		second: {total: %d, slots: [2]int32{%d, %d}},
	}
	alias := values
	current, present := alias[compositeKey%d{left: first.left, right: first.right}]
	if !present { return -1000 }
	current.total += %d
	current.slots[1] ^= %d
	alias[first] = current
	delete(alias, second)
	missing, found := values[second]
	if found { return missing.total - 2000 }
	return values[first].total*3 + int64(values[first].slots[0])*5 + int64(values[first].slots[1])*7 + int64(len(values))*11
}
`, index, index, index, index, random.small(), random.small(), random.small(), index, random.small(), random.small(), random.small(), index, index, random.small(), random.small(), random.small(), random.small(), random.small(), random.small(), index, random.small(), random.small())
}

func interfaceMapCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type interfaceMapKey%d struct { code int32; pair [2]int16 }
func case%d() int64 {
	firstValue := interfaceMapKey%d{code: %d, pair: [2]int16{%d, %d}}
	secondValue := [2]int32{%d, %d}
	var first any = firstValue
	var second any = secondValue
	values := map[any]int64{first: %d, second: %d}
	alias := values
	var equalFirst any = interfaceMapKey%d{code: firstValue.code, pair: firstValue.pair}
	alias[equalFirst] += %d
	current, present := values[first]
	if !present { return -1000 }
	delete(alias, second)
	_, removed := values[secondValue]
	if removed { return -2000 }
	return current*5 + int64(len(values))*13
}
`, index, index, index, random.small(), random.small(), random.small(), random.small(), random.small(), random.small(), random.small(), index, random.small())
}

func mapRangeMutationCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	values := map[int]int64{1: %d, 2: %d, 3: %d, 4: %d}
	total := int64(0)
	mask := int64(0)
	for key, value := range values {
		values[key] = value + int64(key)*%d
		total += values[key]*17 + int64(key)*23
		mask |= int64(1) << uint(key)
		delete(values, key)
	}
	return total + mask*31 + int64(len(values))*47
}
`, index, random.small(), random.small(), random.small(), random.small(), random.next()%9+1)
}

func mapOperandOrderCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	trace := int64(0)
	key := func(tag int64) int { trace = trace*10 + tag; return int(tag) }
	value := func(tag, result int64) int64 { trace = trace*10 + tag; return result }
	values := map[int]int64{1: %d, 2: %d}
	values[key(1)], values[key(2)] = value(3, %d), value(4, %d)
	values[key(5)] += value(6, %d)
	return trace*17 + values[1]*19 + values[2]*23 + values[5]*29
}
`, index, random.small(), random.small(), random.small(), random.small(), random.small())
}

func floatMapCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	first := float64(%d) / 4.0
	second := float64(%d) / 4.0
	equalFirst := first + float64(0)
	values := map[float64]int64{first: %d, second: %d}
	values[equalFirst] += %d
	current, present := values[first]
	if !present { return -1000 }
	delete(values, second)
	return current*7 + int64(len(values))*13
}
`, index, random.small(), random.small(), random.small(), random.small(), random.small())
}

func complexMapCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	first := complex(float64(%d), float64(%d))
	second := complex(float64(%d), float64(%d))
	equalFirst := complex(real(first), imag(first))
	values := map[complex128]int64{first: %d, second: %d}
	values[equalFirst] += %d
	current, present := values[first]
	if !present { return -1000 }
	delete(values, second)
	return current*5 + int64(len(values))*17
}
`, index, random.small(), random.small(), random.small(), random.small(), random.small(), random.small(), random.small())
}

func pointerMapCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	firstValue := int64(%d)
	secondValue := int64(%d)
	first := &firstValue
	second := &secondValue
	values := map[*int64]int64{first: %d, second: %d}
	alias := first
	values[alias] += %d
	current, present := values[first]
	if !present { return -1000 }
	delete(values, second)
	return current*7 + *first*11 + int64(len(values))*13
}
`, index, random.small(), random.small(), random.small(), random.small(), random.small())
}

func nestedCompositeFlowCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type nestedInner%d struct { data [2]int32 }
type nestedOuter%d struct { inner nestedInner%d; rows [2][2]int16 }
func nestedPair%d(value nestedOuter%d) (nestedOuter%d, bool) { return value, true }
func case%d() int64 {
	source := nestedOuter%d{
		inner: nestedInner%d{data: [2]int32{%d, %d}},
		rows: [2][2]int16{{%d, %d}, {%d, %d}},
	}
	result, present := nestedPair%d(nestedOuter%d{inner: source.inner, rows: source.rows})
	if !present { return -1000 }
	return int64(result.inner.data[0])*3 + int64(result.inner.data[1])*5 + int64(result.rows[0][0])*7 + int64(result.rows[1][1])*11
}
`, index, index, index, index, index, index, index, index, index, random.small(), random.small(), random.small(), random.small(), random.small(), random.small(), index, index)
}

func aggregateAssignmentCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type aggregateAssignment%d struct { words [2]int64; text string; number complex128 }
func aggregateIdentity%d(value aggregateAssignment%d) aggregateAssignment%d { return value }
func case%d() int64 {
	left := aggregateAssignment%d{words: [2]int64{%d, %d}, text: "left", number: complex(%d, %d)}
	right := aggregateAssignment%d{words: [2]int64{%d, %d}, text: "right", number: complex(%d, %d)}
	left, right = aggregateIdentity%d(right), aggregateIdentity%d(left)
	pointer := &left
	pointer.words[1] += right.words[0]
	return left.words[0]*3 + left.words[1]*5 + right.words[0]*7 + int64(len(left.text))*11 + int64(real(pointer.number))*13 + int64(imag(pointer.number))*17
}
`, index, index, index, index, index, index, random.small(), random.small(), random.small(), random.small(), index, random.small(), random.small(), random.small(), random.small(), index, index)
}

func mapCompositeValueCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type mapCompositeValue%d struct { pair [2]int32; text string; number complex128 }
func mapCompositeIdentity%d(value mapCompositeValue%d) mapCompositeValue%d { return value }
func case%d() int64 {
	values := map[int]mapCompositeValue%d{
		1: {pair: [2]int32{%d, %d}, text: "first", number: complex(%d, %d)},
		2: {pair: [2]int32{%d, %d}, text: "second", number: complex(%d, %d)},
	}
	value, present := values[1]
	if !present { return -1000 }
	value = mapCompositeIdentity%d(value)
	value.pair[0] += values[2].pair[1]
	values[1] = value
	delete(values, 2)
	result := values[1]
	return int64(result.pair[0])*3 + int64(result.pair[1])*5 + int64(len(result.text))*7 + int64(real(result.number))*11 + int64(imag(result.number))*13 + int64(len(values))*17
}
`, index, index, index, index, index, index, random.small(), random.small(), random.small(), random.small(), random.small(), random.small(), random.small(), random.small(), index)
}

func interfaceComparisonCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type interfaceComparable%d struct { code int32; pair [2]int16; text string }
func case%d() int64 {
	value := interfaceComparable%d{code: %d, pair: [2]int16{%d, %d}, text: "same"}
	var first any = value
	var equal any = interfaceComparable%d{code: value.code, pair: value.pair, text: value.text}
	var different any = interfaceComparable%d{code: value.code, pair: [2]int16{value.pair[0], value.pair[1] + 1}, text: value.text}
	result := int64(0)
	if first == equal { result += 3 }
	if first != different { result += 5 }
	var pointer *interfaceComparable%d
	var typedNil any = pointer
	if typedNil != nil { result += 7 }
	if typedNil == any(pointer) { result += 11 }
	return result
}
`, index, index, index, random.small(), random.small(), random.small(), index, index, index)
}

func complex64FlowCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func complex64Identity%d(value complex64) complex64 { return value }
func case%d() int64 {
	values := [2]complex64{complex(float32(%d), float32(%d)), complex(float32(%d), float32(%d))}
	first := complex64Identity%d(values[0])
	second := complex(real(values[1]), -imag(values[1]))
	values[0], values[1] = second, first
	return int64(real(values[0]))*3 + int64(imag(values[0]))*5 + int64(real(values[1]))*7 + int64(imag(values[1]))*11
}
`, index, index, random.small(), random.small(), random.small(), random.small(), index)
}

func appendOverlapCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	values := []int64{%d, %d, %d, %d, %d}
	result := append(values[1:2], values[0:3]...)
	copy(result[0:3], result[1:4])
	return result[0]*3 + result[1]*5 + result[2]*7 + result[3]*11 + int64(len(result))*13 + int64(cap(result))*17
}
`, index, random.small(), random.small(), random.small(), random.small(), random.small())
}

func closureAggregateCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type closureAggregate%d struct { values [2]int64; text string }
func case%d() int64 {
	state := closureAggregate%d{values: [2]int64{%d, %d}, text: "capture"}
	pointer := &state
	apply := func(delta int64) closureAggregate%d {
		pointer.values[0] += delta
		state.values[1] ^= pointer.values[0]
		return state
	}
	first := apply(%d)
	second := apply(%d)
	return first.values[0]*3 + first.values[1]*5 + second.values[0]*7 + second.values[1]*11 + int64(len(second.text))*13
}
`, index, index, index, random.small(), random.small(), index, random.small(), random.small())
}

func deferPanicChainCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func deferPanicInner%d(value int64) (result int64) {
	defer func() { result = result*3 + %d }()
	defer func() {
		if recovered := recover(); recovered != nil { result += recovered.(int64) }
	}()
	result = value * 5
	panic(value + %d)
}
func case%d() int64 { return deferPanicInner%d(%d) }
`, index, random.small(), random.small(), index, index, random.small())
}

func interfaceRoundTripCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type roundTripInterface%d interface { Apply(int64) int64 }
type roundTripValue%d struct { base int64; pair [2]int32 }
func (value *roundTripValue%d) Apply(delta int64) int64 { value.base += delta + int64(value.pair[0]); return value.base }
func roundTripIdentity%d(value roundTripInterface%d) roundTripInterface%d { return value }
func case%d() int64 {
	value := &roundTripValue%d{base: %d, pair: [2]int32{%d, %d}}
	dynamic := roundTripIdentity%d(value)
	method := dynamic.Apply
	first := method(%d)
	second := dynamic.Apply(%d)
	concrete := dynamic.(*roundTripValue%d)
	return first*3 + second*5 + concrete.base*7 + int64(concrete.pair[1])*11
}
`, index, index, index, index, index, index, index, index, random.small(), random.small(), random.small(), index, random.small(), random.small(), index)
}

func stringRuneRoundTripCase(index int, random *randomSource) string {
	letter := rune('A' + random.next()%26)
	return fmt.Sprintf(`
func case%d() int64 {
	text := "A¢日🙂Z"
	runes := []rune(text)
	runes[0] = %d
	changed := string(runes)
	total := int64(len(changed))*3 + int64(len(runes))*5
	for offset, value := range changed { total += int64(offset)*7 + int64(value) }
	return total
}
`, index, letter)
}

func structPointerFlowCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type structPointerFlow%d struct { pair [2]int64; text string }
func structPointerReturn%d(value *structPointerFlow%d) structPointerFlow%d { return *value }
func case%d() int64 {
	value := structPointerFlow%d{pair: [2]int64{%d, %d}, text: "pointer"}
	alias := &value
	result := structPointerReturn%d(alias)
	alias.pair[0] += %d
	return result.pair[0]*3 + result.pair[1]*5 + alias.pair[0]*7 + int64(len(result.text))*11
}
`, index, index, index, index, index, index, random.small(), random.small(), index, random.small())
}

func complexDivisionCase(index int, random *randomSource) string {
	divisor := int64(1 << (random.next() % 3))
	realPart := random.small() * divisor
	imagPart := random.small() * divisor
	return fmt.Sprintf(`
func case%d() int64 {
	value := complex(float64(%d), float64(%d))
	divisor := complex(float64(%d), float64(0))
	result := value / divisor
	return int64(real(result))*3 + int64(imag(result))*5
}
`, index, realPart, imagPart, divisor)
}

func switchInitializerVariantsCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func case%d() int64 {
	result := int64(0)
	switch value := int64(%d); value & 3 {
	case 0: result += value*3 + 1
	case 1, 2: result += value*5 + 2
	default: result += value*7 + 3
	}
	var dynamic any = int64(%d)
	switch marker := int64(%d); selected := dynamic.(type) {
	case int64: result += marker*11 + selected*13
	default: result = -1000
	}
	return result
}
`, index, random.small(), random.small(), random.small())
}

func emiDeadCodeCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
type emiRecord%d struct { pair [2]int64; text string }
func emiGuard%d(value, expected int64) bool { return value == expected }
func case%d() int64 {
	input := int64(%d)
	result := input*7 + %d
	if emiGuard%d(input, input+1) {
		values := map[int]emiRecord%d{
			1: {pair: [2]int64{%d, %d}, text: "dead"},
			2: {pair: [2]int64{%d, %d}, text: "branch"},
		}
		for key, value := range values {
			result += int64(key)*value.pair[0] + value.pair[1] + int64(len(value.text))
		}
		var dynamic any = values[1]
		result += dynamic.(emiRecord%d).pair[0]
		apply := func(value int64) int64 { result ^= value; return result }
		result = apply(int64(real(complex(float64(input), float64(result)))))
	}
	if emiGuard%d(input, input) {
		preserved := result
		result = (result + input) - input
		if result != preserved { result = -1000 }
	}
	return result
}
`, index, index, index, random.small(), random.small(), index, index, random.small(), random.small(), random.small(), random.small(), index, index)
}

func metamorphicExpressionCase(index int, random *randomSource) string {
	return fmt.Sprintf(`
func metamorphicIdentity%d(value int64) int64 { return value }
func case%d() int64 {
	base := int64(%d)
	variants := [6]int64{
		base,
		base + 0,
		-(-base),
		base ^ 0,
		metamorphicIdentity%d(base),
		func(value int64) int64 { temporary := value; return temporary }(base),
	}
	result := int64(%d)
	for position, value := range variants {
		if value != base { result ^= int64(1) << position }
		result = result*3 + value + int64(position)
	}
	return result
}
`, index, index, random.small(), index, random.small())
}

func policyDataFlowCase(index int, random *randomSource) string {
	var body strings.Builder
	for step := 1; step <= random.policy.dataFlowSteps; step++ {
		fmt.Fprintf(&body, "\tvalue%d := policyFlowStep%d(value%d, int64(%d))\n", step, index, step-1, random.small())
	}
	last := random.policy.dataFlowSteps
	return fmt.Sprintf(`
type policyFlow%d struct { pair [2]int64; text string }
func policyFlowStep%d(value policyFlow%d, delta int64) policyFlow%d {
	value.pair[0] = value.pair[0]*3 + delta
	value.pair[1] = (value.pair[1] ^ value.pair[0]) - delta
	if value.pair[0]&1 == 0 { value.text += "x" }
	return value
}
func case%d() int64 {
	value0 := policyFlow%d{pair: [2]int64{%d, %d}, text: "flow"}
%s	return value%d.pair[0]*3 + value%d.pair[1]*5 + int64(len(value%d.text))*7 + value0.pair[0]*11
}
`, index, index, index, index, index, index, random.small(), random.small(), body.String(), last, last, last)
}

func policyAliasCase(index int, random *randomSource) string {
	var initial strings.Builder
	for position := 0; position < 12; position++ {
		if position != 0 {
			initial.WriteString(", ")
		}
		fmt.Fprintf(&initial, "%d", random.small())
	}
	var body strings.Builder
	for alias := 0; alias < random.policy.aliasCount; alias++ {
		start := int(random.next() % 9)
		position := int(random.next() % 4)
		fmt.Fprintf(&body, "\talias%d := base[%d:%d]\n", alias, start, start+4)
		fmt.Fprintf(&body, "\tpointer%d := &alias%d[%d]\n", alias, alias, position)
		fmt.Fprintf(&body, "\t*pointer%d = *pointer%d*3 + int64(%d)\n", alias, alias, random.small())
		if alias&1 != 0 {
			fmt.Fprintf(&body, "\tcopy(alias%d[1:], alias%d[:3])\n", alias, alias)
		}
	}
	return fmt.Sprintf(`
func case%d() int64 {
	values := [12]int64{%s}
	base := values[:]
%s	result := int64(0)
	for position, value := range values { result = result*5 + int64(position)*3 + value }
	return result + int64(len(base))*7 + int64(cap(base))*11
}
`, index, initial.String(), body.String())
}

func policyMaterializationCase(index int, random *randomSource) string {
	left, right, child := random.small(), random.small(), random.small()
	literal := fmt.Sprintf("policyMaterial%d{pair: [2]int64{%d, %d}, child: policyMaterialChild%d{value: %d}, text: \"material\"}", index, left, right, index, child)
	mode := random.policy.materialization
	if mode < 0 {
		mode = int(random.next() % 7)
	}
	var setup string
	valueExpression := "value"
	switch mode {
	case 0:
		valueExpression = "(" + literal + ")"
	case 1:
		setup = fmt.Sprintf("\tvalue := %s\n", literal)
	case 2:
		valueExpression = fmt.Sprintf("policyMaterialIdentity%d(policyMaterialIdentity%d(%s))", index, index, literal)
	case 3:
		setup = fmt.Sprintf("\tvar dynamic any = %s\n", literal)
		valueExpression = fmt.Sprintf("dynamic.(policyMaterial%d)", index)
	case 4:
		valueExpression = fmt.Sprintf("([1]policyMaterial%d{%s}[0])", index, literal)
	case 5:
		setup = fmt.Sprintf("\tpointer := &%s\n", literal)
		valueExpression = "pointer"
	default:
		setup = fmt.Sprintf("\tmapping := map[int]policyMaterial%d{0: %s}\n", index, literal)
		valueExpression = "mapping[0]"
	}
	body := fmt.Sprintf("%s\tresult := %s.pair[0]*3 + %s.pair[1]*5 + %s.child.value*7 + int64(len(%s.text))*11\n", setup, valueExpression, valueExpression, valueExpression, valueExpression)
	return fmt.Sprintf(`
type policyMaterialChild%d struct { value int64 }
type policyMaterial%d struct { pair [2]int64; child policyMaterialChild%d; text string }
func policyMaterialIdentity%d(value policyMaterial%d) policyMaterial%d { return value }
func policyMaterialConsume%d(value policyMaterial%d) int64 {
	return value.pair[0]*3 + value.pair[1]*5 + value.child.value*7 + int64(len(value.text))*11
}
func case%d() int64 {
%s	return result
}
`, index, index, index, index, index, index, index, index, index, body)
}
