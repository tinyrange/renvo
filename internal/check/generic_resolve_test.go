package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestGenericTypeIdentityTagsAndArrayConstants(t *testing.T) {
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" +
		"const Width = 2+2\n" +
		"type A = struct { X int `json:\"x\"` }\n" +
		"type B = struct { X int \"json:\\\"x\\\"\" }\n" +
		"type C = struct { X int `json:\"y\"` }\n" +
		"type Array[T any] [Width*2]T\n")}})
	e := newGenericEnvironment(&graph)
	for i := range e.decls {
		e.resolveDeclaration(i)
	}
	if e.errorText != "" {
		t.Fatal(e.errorText)
	}
	root := checkRootPackage(t, graph)
	a, b, c := e.decls[e.lookup(root, "A")].typ, e.decls[e.lookup(root, "B")].typ, e.decls[e.lookup(root, "C")].typ
	if a != b || a == c {
		t.Fatal("struct tag identities were lost")
	}
	array := e.types.get(e.types.underlying(e.decls[e.lookup(root, "Array")].typ))
	if array.length != 8 {
		t.Fatalf("array length: %d", array.length)
	}
	if got := genericQuoted(e.types.get(a).fields[0].tag); got != `"json:\"x\""` {
		t.Fatalf("tag source: %s", got)
	}
}

func TestGenericResolveDeclarations(t *testing.T) {
	graph := genericTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
import "example.com/case/lib"
type Numbers []int
type Alias = lib.Node[int]
func Clone[S ~[]E, E any](s S) S { return s }
func Choose[T lib.Number](a, b T) T { return a }
func main() {}
`)},
		{Path: "/repo/case/lib/lib.go", Src: []byte(`package lib
type Number interface { ~int | ~int64 }
type Node[T any] struct { Value T; Next *Node[T] }
type List[T any] = []T
`)},
	})
	e := newGenericEnvironment(&graph)
	for i := range e.decls {
		e.resolveDeclaration(i)
	}
	if e.errorText != "" {
		t.Fatalf("%s at %d/%d/%d", e.errorText, e.errorPkg, e.errorFile, e.errorToken)
	}
	root := checkRootPackage(t, graph)
	clone := e.decls[e.lookup(root, "Clone")]
	ints := e.decls[e.lookup(root, "Numbers")].typ
	args, ok := e.types.infer(clone.parameters, clone.constraints, clone.typ, nil, []genericArgument{{typ: ints}}, false, 0)
	if !ok || len(args) != 2 || args[0] != ints || args[1] != e.types.basic("int") {
		t.Fatalf("Clone inference: %v, %v", args, ok)
	}
	choose := e.decls[e.lookup(root, "Choose")]
	if !e.types.satisfies(e.types.basic("int64"), choose.constraints[0]) || e.types.satisfies(e.types.basic("string"), choose.constraints[0]) {
		t.Fatal("imported constraint not resolved")
	}
	alias := e.decls[e.lookup(root, "Alias")].typ
	node := e.types.get(alias)
	if node.name != "Node" || node.args[0] != e.types.basic("int") {
		t.Fatalf("alias identity: %#v", node)
	}
	body := e.types.get(node.underlying)
	if len(body.fields) != 2 || body.fields[0].typ != e.types.basic("int") || e.types.get(body.fields[1].typ).elem != alias {
		t.Fatalf("recursive instantiation: %#v", body)
	}
}

func TestGenericResolveInvalidConstraints(t *testing.T) {
	for _, decl := range []string{
		"func F[T any, T any]() {}",
		"func F[T interface{ int | ~int }]() {}",
		"type Count int; func F[T ~Count]() {}",
		"func F[T interface{ Read() int; Read() string }]() {}",
		"type A interface{ Read() int }; type B interface{ Read() string }; func F[T interface{ A; B }]() {}",
		"type A[T any] = A[T]",
		"type A[T any] struct{ Next *A[[]T] }",
	} {
		t.Run(decl, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main; " + decl)}})
			e := newGenericEnvironment(&graph)
			for i := range e.decls {
				e.resolveDeclaration(i)
			}
			e.checkInstantiationCycles()
			if e.errorText == "" {
				t.Fatal("accepted invalid generic declaration")
			}
		})
	}
}

func TestGenericInstantiationCycleReachability(t *testing.T) {
	for _, tc := range []struct {
		name    string
		edges   []genericInstantiationEdge
		invalid bool
	}{
		{name: "finite cycles", edges: []genericInstantiationEdge{
			{from: 1, to: 2}, {from: 2, to: 1},
			{from: 3, to: 4, growing: true}, {from: 4, to: 5},
		}},
		{name: "separate growing walks", invalid: true, edges: []genericInstantiationEdge{
			{from: 1, to: 3, growing: true},
			{from: 2, to: 3, growing: true}, {from: 3, to: 2},
		}},
		{name: "growing self edge", invalid: true, edges: []genericInstantiationEdge{
			{from: 4, to: 4, growing: true},
		}},
		{name: "branch reaches growing cycle", invalid: true, edges: []genericInstantiationEdge{
			{from: 1, to: 2, growing: true}, {from: 2, to: 3},
			{from: 2, to: 4}, {from: 4, to: 5}, {from: 5, to: 1},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := genericEnvironment{types: genericTypes{items: make([]genericType, 6)}, edges: tc.edges}
			e.checkInstantiationCycles()
			if (e.errorText != "") != tc.invalid {
				t.Fatalf("cycle diagnostic = %q, want invalid %v", e.errorText, tc.invalid)
			}
		})
	}
}
