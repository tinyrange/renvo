package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestGenericSemanticTypeDisplay(t *testing.T) {
	graph := genericTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/lib/lib.go", Src: []byte(`package lib
type Count int
type Box[T any] struct{Value T}
`)},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
import "example.com/case/lib"
type Local int
type Alias = lib.Count
type Named = lib.Box[lib.Count]
type LocalArg = lib.Box[Local]
type Empty = lib.Box[struct{}]
type Any = lib.Box[any]
type NamedError = lib.Box[error]
type ErrorShape = lib.Box[interface{Error()string}]
type Embedded = lib.Box[struct{x int;Alias;V string "json:\"v\""}]
type Methods = lib.Box[interface{Z(int,...string)(int,string);A();a()}]
type Composite = lib.Box[map[string][]*lib.Count]
func main(){}
`)},
	})
	e := newGenericEnvironment(&graph)
	for i := range e.decls {
		e.resolveDeclaration(i)
	}
	if e.errorText != "" {
		t.Fatal(e.errorText)
	}
	root := checkRootPackage(t, graph)
	for _, test := range []struct{ name, want string }{
		{"Named", "Box[example.com/case/lib.Count]"},
		{"LocalArg", "Box[main.Local]"},
		{"Empty", "Box[struct {}]"},
		{"Any", "Box[interface {}]"},
		{"NamedError", "Box[error]"},
		{"ErrorShape", "Box[interface { Error() string }]"},
		{"Embedded", "Box[struct { main.x int; Alias = example.com/case/lib.Count; V string \"json:\\\"v\\\"\" }]"},
		{"Methods", "Box[interface { A(); Z(int, ...string) (int, string); main.a() }]"},
		{"Composite", "Box[map[string][]*example.com/case/lib.Count]"},
	} {
		id := e.decls[e.lookup(root, test.name)].typ
		if got := e.typeDisplay(id, false); got != test.want {
			t.Errorf("%s: %q, want %q", test.name, got, test.want)
		}
	}
	named := e.types.get(e.decls[e.lookup(root, "NamedError")].typ).args[0]
	shape := e.types.get(e.decls[e.lookup(root, "ErrorShape")].typ).args[0]
	if named == shape || e.types.underlying(named) != shape {
		t.Fatal("predeclared error lost its defined type identity")
	}
	pointer := e.types.intern(genericType{kind: genericPointer, elem: named})
	if len(e.types.methodSet(named)) != 1 || len(e.types.methodSet(pointer)) != 0 {
		t.Fatal("predeclared error has an invalid method set")
	}
}
