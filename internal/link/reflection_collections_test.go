package link

import (
	"renvo.dev/internal/unit"
	"testing"
)

func TestReflectionCollectionAliasIdentity(t *testing.T) {
	source := []byte("package main\ntype Alias = byte\ntype Defined []int\nfunc main() {}\n")
	program := unit.Program{Package: "main"}
	if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
		t.Fatal("parse")
	}
	for _, typ := range []string{"[]byte", "[]uint8", "[]Alias"} {
		if got := reflectionCanonicalType(&program, typ, false); got != "[]uint8" {
			t.Fatalf("%s identity = %s", typ, got)
		}
	}
	if reflectionCanonicalType(&program, "Defined", false) != "Defined" || reflectionCanonicalType(&program, "Defined", true) != "[]int" {
		t.Fatal("defined slice identity")
	}
}
