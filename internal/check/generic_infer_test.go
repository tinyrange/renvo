package check

import "testing"

func TestGenericInferenceRejectsRecursiveEquations(t *testing.T) {
	var types genericTypes
	p := types.intern(genericType{kind: genericParameter, name: "T", origin: "recursive/0"})
	q := types.intern(genericType{kind: genericParameter, name: "U", origin: "recursive/1"})
	sliceQ := types.intern(genericType{kind: genericSlice, elem: q})
	pointerP := types.intern(genericType{kind: genericPointer, elem: p})
	s := genericInference{types: &types, parameters: []int{p, q}, arguments: make([]int, 2)}
	if !s.unify(p, sliceQ, false) {
		t.Fatal("rejected finite partial equation")
	}
	if s.unify(q, pointerP, false) {
		t.Fatal("accepted recursive equation")
	}
	if !s.unify(q, types.basic("int"), false) {
		t.Fatal("rejected finite solution")
	}
}

func TestGenericInferenceArguments(t *testing.T) {
	var types genericTypes
	p := types.intern(genericType{kind: genericParameter, name: "T", origin: "F/0"})
	fn := types.intern(genericType{kind: genericFunc, params: []int{p, p}, results: []int{p}})
	integer, str := types.basic("int"), types.basic("string")
	named := types.intern(genericType{kind: genericNamed, name: "Count", origin: "a.Count", underlying: integer})
	for _, tc := range []struct {
		name     string
		explicit []int
		actual   []genericArgument
		want     int
	}{
		{"typed", nil, []genericArgument{{typ: integer}, {typ: integer}}, integer},
		{"named_and_constant", nil, []genericArgument{{typ: named}, {untyped: genericUntypedInt}}, named},
		{"constant_and_named", nil, []genericArgument{{untyped: genericUntypedInt}, {typ: named}}, named},
		{"mixed_untyped", nil, []genericArgument{{untyped: genericUntypedInt}, {untyped: genericUntypedFloat}}, types.basic("float64")},
		{"rune_default", nil, []genericArgument{{untyped: genericUntypedInt}, {untyped: genericUntypedRune}}, types.basic("int32")},
		{"explicit", []int{named}, []genericArgument{{untyped: genericUntypedInt}, {untyped: genericUntypedInt}}, named},
		{"conflicting", nil, []genericArgument{{typ: integer}, {typ: str}}, 0},
		{"distinct_named", nil, []genericArgument{{typ: integer}, {typ: named}}, 0},
		{"distinct_named_reverse", nil, []genericArgument{{typ: named}, {typ: integer}}, 0},
		{"nil_cannot_infer", nil, []genericArgument{{untyped: genericUntypedNil}, {untyped: genericUntypedNil}}, 0},
		{"incompatible_untyped", nil, []genericArgument{{untyped: genericUntypedString}, {untyped: genericUntypedInt}}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := types.infer([]int{p}, []genericConstraint{{all: true}}, fn, tc.explicit, tc.actual, false, 0)
			if ok != (tc.want != 0) || ok && (len(got) != 1 || got[0] != tc.want) {
				t.Fatalf("inferred %v, %v; want %d", got, ok, tc.want)
			}
		})
	}
}

func TestGenericInferenceDependentConstraints(t *testing.T) {
	var types genericTypes
	s := types.intern(genericType{kind: genericParameter, name: "S", origin: "Clone/0"})
	e := types.intern(genericType{kind: genericParameter, name: "E", origin: "Clone/1"})
	sliceE := types.intern(genericType{kind: genericSlice, elem: e})
	fn := types.intern(genericType{kind: genericFunc, params: []int{s}, results: []int{s}})
	integer := types.basic("int")
	sliceInt := types.intern(genericType{kind: genericSlice, elem: integer})
	named := types.intern(genericType{kind: genericNamed, name: "Ints", origin: "a.Ints", underlying: sliceInt})
	constraints := []genericConstraint{{terms: []genericTerm{{typ: sliceE, tilde: true}}}, {all: true}}
	for _, explicit := range [][]int{nil, {named}} {
		got, ok := types.infer([]int{s, e}, constraints, fn, explicit, []genericArgument{{typ: named}}, false, 0)
		if !ok || !genericIDsEqual(got, []int{named, integer}) {
			t.Fatalf("dependent inference: %v, %v", got, ok)
		}
	}
	if _, ok := types.infer([]int{s, e}, constraints, fn, []int{named, types.basic("string")}, []genericArgument{{typ: named}}, false, 0); ok {
		t.Fatal("accepted contradictory explicit argument")
	}
}

func TestGenericInferenceFunctionValuesAndVariadics(t *testing.T) {
	var types genericTypes
	p := types.intern(genericType{kind: genericParameter, name: "T", origin: "Identity/0"})
	integer := types.basic("int")
	fn := types.intern(genericType{kind: genericFunc, params: []int{p}, results: []int{p}})
	expected := types.intern(genericType{kind: genericFunc, params: []int{integer}, results: []int{integer}})
	got, ok := types.infer([]int{p}, []genericConstraint{{all: true}}, fn, nil, nil, false, expected)
	if !ok || got[0] != integer {
		t.Fatal("function value context did not infer T")
	}
	sliceP := types.intern(genericType{kind: genericSlice, elem: p})
	sliceInt := types.intern(genericType{kind: genericSlice, elem: integer})
	variadic := types.intern(genericType{kind: genericFunc, params: []int{sliceP}, variadic: true})
	got, ok = types.infer([]int{p}, []genericConstraint{{all: true}}, variadic, nil, []genericArgument{{untyped: genericUntypedInt}, {untyped: genericUntypedInt}}, false, 0)
	if !ok || got[0] != integer {
		t.Fatal("variadic elements did not infer T")
	}
	got, ok = types.infer([]int{p}, []genericConstraint{{all: true}}, variadic, nil, []genericArgument{{typ: sliceInt}}, true, 0)
	if !ok || got[0] != integer {
		t.Fatal("variadic spread did not infer T")
	}
	if _, ok = types.infer([]int{p}, []genericConstraint{{all: true}}, variadic, nil, nil, false, 0); ok {
		t.Fatal("empty variadic call inferred unconstrained T")
	}
}

func TestGenericInferenceMethodConstraint(t *testing.T) {
	var types genericTypes
	v := types.intern(genericType{kind: genericParameter, name: "V", origin: "Read/0"})
	e := types.intern(genericType{kind: genericParameter, name: "E", origin: "Read/1"})
	integer := types.basic("int")
	method := types.intern(genericType{kind: genericFunc, results: []int{e}})
	actualMethod := types.intern(genericType{kind: genericFunc, results: []int{integer}})
	reader := types.intern(genericType{kind: genericNamed, name: "Reader", origin: "a.Reader", underlying: integer, methods: []genericMethod{{name: "Read", typ: actualMethod}}})
	fn := types.intern(genericType{kind: genericFunc, params: []int{v}, results: []int{e}})
	constraints := []genericConstraint{{all: true, methods: []genericMethod{{name: "Read", typ: method}}}, {all: true}}
	got, ok := types.infer([]int{v, e}, constraints, fn, nil, []genericArgument{{typ: reader}}, false, 0)
	if !ok || !genericIDsEqual(got, []int{reader, integer}) {
		t.Fatalf("method inference: %v, %v", got, ok)
	}
}
