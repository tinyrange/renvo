package main

type identityNamedInt int
type identityOtherInt int

type identityPair struct {
	a int
	b string
}

type identityMethod interface {
	Identity() int
}

type identityValue struct {
	value int
}

func (v identityValue) Identity() int {
	return v.value
}

type identityPointer struct {
	value int
}

func (v *identityPointer) Identity() int {
	if v == nil {
		return 0
	}
	return v.value
}

type identityHolder struct {
	value interface{}
}

var identityGlobal interface{}

func identityPass(value interface{}) interface{} {
	return value
}

func identityPassMethod(value identityMethod) identityMethod {
	return value
}

func identityVariadic(values ...interface{}) bool {
	return values[0] == values[1]
}

func identityNonComparablePanics() (ok bool) {
	defer func() {
		ok = recover() != nil
	}()
	var left interface{} = []int{1}
	var right interface{} = []int{1}
	return left == right
}

func appMain() int {
	var pointer *identityPointer
	var method identityMethod = pointer
	var empty interface{}
	if method == nil || identityPass(method) == nil || identityPassMethod(method) == nil || empty != nil {
		return 1
	}

	var valueMethod identityMethod = identityValue{value: 9}
	var pointerMethod identityMethod = &identityPointer{value: 10}
	if identityPassMethod(valueMethod).Identity() != 9 || identityPassMethod(pointerMethod).Identity() != 10 {
		return 2
	}

	var one interface{} = identityNamedInt(7)
	var same interface{} = identityNamedInt(7)
	var changed interface{} = identityNamedInt(8)
	var other interface{} = identityOtherInt(7)
	if one != same || one == changed || one == other {
		return 3
	}

	var stringLeft interface{} = "same text"
	var stringRight interface{} = "same text"
	var pairLeft interface{} = identityPair{a: 3, b: "pair"}
	var pairRight interface{} = identityPair{a: 3, b: "pair"}
	var pairOther interface{} = identityPair{a: 4, b: "pair"}
	if stringLeft != stringRight || pairLeft != pairRight || pairLeft == pairOther {
		return 4
	}

	array := [2]int{5, 6}
	var arrayLeft interface{} = array
	var arrayRight interface{} = [2]int{5, 6}
	if arrayLeft != arrayRight {
		return 5
	}

	identityGlobal = pairLeft
	holder := identityHolder{value: identityGlobal}
	values := []interface{}{holder.value, pairRight}
	lookup := map[string]interface{}{"value": pairLeft}
	if identityGlobal != pairRight || holder.value != pairRight || values[0] != values[1] || lookup["value"] != pairRight {
		return 6
	}

	captured := pairLeft
	closure := func() bool {
		return captured == pairRight
	}
	if !closure() || !identityVariadic(pairLeft, pairRight) || !identityNonComparablePanics() {
		return 7
	}

	print("PASS\n")
	return 0
}
