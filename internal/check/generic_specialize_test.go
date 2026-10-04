package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
	"strings"
	"testing"
)

func TestGenericSpecializationConcreteDeclarations(t *testing.T) {
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type Node[T any] struct { Value T; Next *Node[T] }
func Identity[T any](value T) T { return value }
func main() {}
`)}})
	e := newGenericEnvironment(&graph)
	s := genericSpecializer{environment: e}
	root := checkRootPackage(t, graph)
	integer := e.types.basic("int")
	identity := e.lookup(root, "Identity")
	index := s.instantiate(identity, []int{integer})
	if index < 0 || s.instantiate(identity, []int{integer}) != index {
		t.Fatal("function instantiation not memoized")
	}
	node := s.instantiate(e.lookup(root, "Node"), []int{integer})
	if node < 0 {
		t.Fatal("type instantiation rejected")
	}
	src := "package main\n" + s.declarationText(index) + s.declarationText(node)
	if strings.Contains(src, "[T any]") || !strings.Contains(src, "type T = int") || !strings.Contains(src, "Next *"+s.instances[node].name) {
		t.Fatalf("bad specialization:\n%s", src)
	}
	parsed := syntax.ParseFile([]byte(src))
	if !parsed.Ok || parsed.Generics != nil {
		t.Fatalf("specialization is not concrete Go:\n%s", src)
	}
	concrete := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(src)}})
	if checked := CheckGraph(concrete); !checked.Ok {
		t.Fatalf("concrete check failed: %d/%d/%d\n%s", checked.Error, checked.ErrorFile, checked.ErrorToken, src)
	}
}

func TestGenericSpecializationRejectsConstraintViolation(t *testing.T) {
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main; func Double[T ~int](v T) T { return v+v }")}})
	e := newGenericEnvironment(&graph)
	s := genericSpecializer{environment: e}
	if s.instantiate(e.lookup(checkRootPackage(t, graph), "Double"), []int{e.types.basic("string")}) >= 0 {
		t.Fatal("specialized invalid type argument")
	}
}
