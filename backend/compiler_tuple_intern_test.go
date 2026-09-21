package main

import "testing"

func TestTupleTypeInterning(t *testing.T) {
	program := renvoParseProgram([]byte("package main\nfunc pair() ([]byte, error) { return nil, nil }\n"))
	meta := renvoBuildMeta(&program)
	if !program.ok || !meta.ok || len(meta.funcs) != 1 {
		t.Fatal("metadata")
	}
	result := meta.funcs[0].resultType
	tuple := renvoResolveType(&meta, result)
	if !renvoTypeIsTuple(&meta, result) {
		t.Fatal("not a tuple")
	}
	types := []int{meta.fields[tuple.first].typ, meta.fields[tuple.first+1].typ}
	typeCount, fieldCount := len(meta.types), len(meta.fields)
	for i := 0; i < 100; i++ {
		if got := renvoBuildTupleType(&meta, types); got != result {
			t.Fatalf("duplicate tuple type: got %d want %d", got, result)
		}
	}
	if len(meta.types) != typeCount || len(meta.fields) != fieldCount {
		t.Fatal("interning grew metadata")
	}
	reversed := renvoBuildTupleType(&meta, []int{types[1], types[0]})
	if reversed == result {
		t.Fatal("different result order merged")
	}
	if renvoTypesEquivalent(&meta, result, reversed) {
		t.Fatal("different result order considered equivalent")
	}
}
