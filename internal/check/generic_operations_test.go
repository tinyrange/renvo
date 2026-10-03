package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestGenericOperationsCheckEntireTypeSet(t *testing.T) {
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
func Numeric[T ~int | ~float64](v T) {}
func Mixed[T ~int | ~string](v T) {}
func Anything[T any](v T) {}
func Comparable[T comparable](v T) {}
func Indexable[T ~[]byte | ~string](v T) {}
func Mismatch[T ~[]int | ~string](v T) {}
type Ordered interface { ~int | ~int64 |
 ~float32 | ~float64 |
 ~string }
func OrderedValue[T Ordered](v T) {}
func main() {}
`)}})
	e := newGenericEnvironment(&graph)
	for i := range e.decls {
		e.resolveDeclaration(i)
	}
	if e.errorText != "" {
		t.Fatal(e.errorText)
	}
	root := checkRootPackage(t, graph)
	for _, tc := range []struct {
		fn, op string
		want   bool
	}{
		{"Numeric", "+", true}, {"Numeric", "%", false}, {"Numeric", "<", true},
		{"Mixed", "+", true}, {"Mixed", "-", false}, {"Mixed", "==", true},
		{"Anything", "+", false}, {"Anything", "==", false}, {"Anything", "nil", false},
		{"Comparable", "==", true}, {"Comparable", "<", false},
		{"OrderedValue", "<", true}, {"OrderedValue", "-", false},
	} {
		p := e.decls[e.lookup(root, tc.fn)].parameters[0]
		if got := e.allowsOperation(p, tc.op); got != tc.want {
			t.Errorf("%s %s = %v", tc.fn, tc.op, got)
		}
	}
	p := e.decls[e.lookup(root, "Indexable")].parameters[0]
	_, elem, ok := e.indexType(p)
	if !ok || elem != e.types.basic("byte") {
		t.Fatal("compatible indexing rejected")
	}
	p = e.decls[e.lookup(root, "Mismatch")].parameters[0]
	if _, _, ok := e.indexType(p); ok {
		t.Fatal("incompatible indexing accepted")
	}
}

func TestGenericConstraintInterfaceUnionOverlap(t *testing.T) {
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type Floats interface { ~float32 | ~float64 }
type Combined interface { float32 | Floats }
type Universal interface { int | any }
func F[T Combined, U Universal]() {}
`)}})
	e := newGenericEnvironment(&graph)
	for i := range e.decls {
		e.resolveDeclaration(i)
	}
	if e.errorText != "" {
		t.Fatal(e.errorText)
	}
	d := e.decls[e.lookup(checkRootPackage(t, graph), "F")]
	if !e.types.satisfies(e.types.basic("float32"), d.constraints[0]) || e.types.satisfies(e.types.basic("int"), d.constraints[0]) || !d.constraints[1].all {
		t.Fatal("incorrect union of interface type sets")
	}
}
