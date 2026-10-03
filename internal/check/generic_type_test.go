package check

import "testing"

func TestGenericTypeIdentity(t *testing.T) {
	var types genericTypes
	integer := types.basic("int")
	if types.basic("byte") != types.basic("uint8") || types.basic("rune") != types.basic("int32") {
		t.Fatal("predeclared aliases lost identity")
	}
	a := types.intern(genericType{kind: genericNamed, name: "Item", origin: "example/a.Item", underlying: integer})
	b := types.intern(genericType{kind: genericNamed, name: "Item", origin: "example/b.Item", underlying: integer})
	if a == b || a == integer || types.underlying(a) != integer {
		t.Fatal("named identity confused with representation")
	}
	first := types.intern(genericType{kind: genericParameter, name: "T", origin: "a.F/0"})
	second := types.intern(genericType{kind: genericParameter, name: "T", origin: "a.G/0"})
	if first == second {
		t.Fatal("parameter scopes were merged")
	}
	args := []int{integer}
	box := types.intern(genericType{kind: genericNamed, name: "Box", origin: "a.Box", args: args})
	args[0] = b
	if types.get(box).args[0] != integer {
		t.Fatal("interning retained scratch arguments")
	}
	for i := 0; i < 200; i++ {
		types.intern(genericType{kind: genericArray, length: uint64(i), elem: integer})
	}
	if got := types.intern(genericType{kind: genericNamed, name: "Item", origin: "example/a.Item"}); got != a {
		t.Fatal("identity changed after table growth")
	}
	left := types.intern(genericType{kind: genericStruct, fields: []genericField{{name: "hidden", pkg: "a", typ: integer}}})
	right := types.intern(genericType{kind: genericStruct, fields: []genericField{{name: "hidden", pkg: "b", typ: integer}}})
	if left == right {
		t.Fatal("unexported fields from different packages are identical")
	}
	if types.intern(genericType{kind: genericChan, elem: integer, direction: ChanBoth}) == types.intern(genericType{kind: genericChan, elem: integer, direction: ChanSendOnly}) {
		t.Fatal("channel direction lost")
	}
}

func TestGenericSubstitutionRecursiveTypes(t *testing.T) {
	var types genericTypes
	p := types.intern(genericType{kind: genericParameter, name: "T", origin: "a.Node/0"})
	node := types.intern(genericType{kind: genericNamed, name: "Node", origin: "a.Node", args: []int{p}})
	pointer := types.intern(genericType{kind: genericPointer, elem: node})
	body := types.intern(genericType{kind: genericStruct, fields: []genericField{{name: "Value", typ: p}, {name: "Next", typ: pointer}}})
	types.items[node-1].underlying = body
	signature := types.intern(genericType{kind: genericFunc, results: []int{p}})
	types.items[node-1].methods = []genericMethod{{name: "ValueOf", typ: signature}}
	integer := types.basic("int")
	concrete := types.substitute(node, []int{p}, []int{integer})
	v := types.get(concrete)
	u := types.get(v.underlying)
	if v.args[0] != integer || u.fields[0].typ != integer || types.get(u.fields[1].typ).elem != concrete {
		t.Fatalf("recursive substitution = %#v / %#v", v, u)
	}
	if types.get(v.methods[0].typ).results[0] != integer {
		t.Fatal("method signature was not substituted")
	}
	if types.get(body).fields[0].typ != p {
		t.Fatal("substitution mutated template")
	}
	if types.substitute(node, []int{p}, []int{integer}) != concrete {
		t.Fatal("instantiation was not deduplicated")
	}
	if types.substitute(node, []int{p}, []int{types.basic("string")}) == concrete {
		t.Fatal("distinct instantiations merged")
	}
}

func TestGenericConstraintTypeSets(t *testing.T) {
	var types genericTypes
	integer, str := types.basic("int"), types.basic("string")
	named := types.intern(genericType{kind: genericNamed, name: "Count", origin: "a.Count", underlying: integer})
	ints, ok := types.union([]genericTerm{{typ: integer, tilde: true}})
	if !ok || !types.satisfies(named, ints) {
		t.Fatal("~int should contain Count")
	}
	exact, _ := types.union([]genericTerm{{typ: integer}})
	if types.satisfies(named, exact) {
		t.Fatal("int should not contain Count")
	}
	for _, terms := range [][]genericTerm{
		{{typ: integer}, {typ: integer, tilde: true}},
		{{typ: named}, {typ: integer, tilde: true}},
		{{typ: named, tilde: true}},
	} {
		if _, ok := types.union(terms); ok {
			t.Fatalf("accepted invalid union: %#v", terms)
		}
	}
	strings, _ := types.union([]genericTerm{{typ: str, tilde: true}})
	empty, ok := types.intersect(ints, strings)
	if !ok || empty.all || len(empty.terms) != 0 || types.satisfies(integer, empty) {
		t.Fatal("intersection should be empty")
	}
	if !types.constraintSubset(empty, exact) || !types.constraintSubset(exact, ints) || types.constraintSubset(ints, exact) {
		t.Fatal("type-set inclusion is incorrect")
	}
	intersection, _ := types.intersect(ints, exact)
	if len(intersection.terms) != 1 || intersection.terms[0].tilde {
		t.Fatal("intersection lost exact term")
	}
}

func TestGenericComparableConstraint(t *testing.T) {
	var types genericTypes
	integer := types.basic("int")
	any := types.intern(genericType{kind: genericInterface})
	slice := types.intern(genericType{kind: genericSlice, elem: integer})
	withInterface := types.intern(genericType{kind: genericStruct, fields: []genericField{{name: "Value", typ: any}}})
	comparable := genericConstraint{all: true, comparable: true}
	if !types.satisfies(any, comparable) || !types.satisfies(withInterface, comparable) {
		t.Fatal("missing Go 1.20 comparable exception")
	}
	if types.comparable(any, true) || types.comparable(withInterface, true) || types.satisfies(slice, comparable) {
		t.Fatal("incorrect strict comparability")
	}
	if types.constraintSubset(genericConstraint{all: true}, comparable) {
		t.Fatal("unconstrained type parameter satisfies comparable")
	}
	terms, _ := types.union([]genericTerm{{typ: integer}, {typ: slice, tilde: true}})
	filtered, _ := types.intersect(terms, comparable)
	if len(filtered.terms) != 1 || filtered.terms[0].typ != integer {
		t.Fatal("comparable failed to filter noncomparable union terms")
	}
}

func TestGenericConstraintMethods(t *testing.T) {
	var types genericTypes
	integer, str := types.basic("int"), types.basic("string")
	intSig := types.intern(genericType{kind: genericFunc, results: []int{integer}})
	strSig := types.intern(genericType{kind: genericFunc, results: []int{str}})
	m := genericMethod{name: "Value", typ: intSig}
	need := genericConstraint{all: true, methods: []genericMethod{m}}
	good := types.intern(genericType{kind: genericNamed, origin: "a.Good", underlying: integer, methods: []genericMethod{m}})
	bad := types.intern(genericType{kind: genericNamed, origin: "a.Bad", underlying: integer, methods: []genericMethod{{name: "Value", typ: strSig}}})
	if !types.satisfies(good, need) || types.satisfies(bad, need) {
		t.Fatal("method signature satisfaction is incorrect")
	}
	if _, ok := types.intersect(need, genericConstraint{all: true, methods: []genericMethod{{name: "Value", typ: strSig}}}); ok {
		t.Fatal("conflicting embedded methods accepted")
	}
	if _, ok := genericMergeMethods([]genericMethod{{name: "hidden", pkg: "a", typ: intSig}}, []genericMethod{{name: "hidden", pkg: "b", typ: strSig}}); !ok {
		t.Fatal("unexported methods from distinct packages conflict")
	}
}
