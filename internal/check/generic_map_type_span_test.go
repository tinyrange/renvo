package check

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"renvo.dev/internal/load"
)

func TestGenericMapTypeReplacementKeepsParentheses(t *testing.T) {
	for _, expression := range []string{
		`make(map[K]V)`,
		`make((map[K]V))`,
		`make(((map[K]V)))`,
		`make((map[K]V), 2)`,
		`(map[K]V)(nil)`,
		`((map[K]V))(nil)`,
		`*new(map[K]V)`,
		`*new((map[K]V))`,
	} {
		t.Run(expression, func(t *testing.T) {
			source := []byte("package main;func New[K comparable,V any]()map[K]V{return " + expression + "};func main(){_=New[string,int]()}\n")
			set := token.NewFileSet()
			parsed, err := parser.ParseFile(set, "main.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			config := types.Config{}
			if _, err := config.Check("example.com/case", set, []*ast.File{parsed}, nil); err != nil {
				t.Fatal(err)
			}
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: source}})
			prepared := PrepareGenerics(graph)
			if !prepared.Ok {
				t.Fatal(prepared.Message)
			}
			if checked := CheckGraphCore(prepared.Graph); !checked.Ok {
				t.Fatalf("concrete error=%d token=%d", checked.Error, checked.ErrorToken)
			}
			importer := genericConcreteImporter{t: t, graph: prepared.Graph, packages: map[string]*types.Package{}}
			importer.Import(graph.Root)
		})
	}
}
