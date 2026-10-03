package check

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"renvo.dev/internal/load"
)

func TestLiteralLabelsHaveIndependentFunctionScopes(t *testing.T) {
	for _, tc := range []struct {
		body  string
		valid bool
	}{
		{`f:=func(){goto Exit;Exit:return};f();goto Exit;Exit:`, true},
		{`goto Exit;Exit:;Exit:`, false},
		{`f:=func(){goto Exit;Exit:;Exit:};f()`, false},
		{`f:=func(){goto Exit;Exit:return};f();goto Exit`, false},
	} {
		source := []byte("package main;func main(){" + tc.body + "}\n")
		set := token.NewFileSet()
		file, err := parser.ParseFile(set, "main.go", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		config := types.Config{}
		_, err = config.Check("example.com/case", set, []*ast.File{file}, nil)
		if (err == nil) != tc.valid {
			t.Fatalf("Go validity: %v", err)
		}
		graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: source}})
		checked := CheckGraphCore(graph)
		if checked.Ok != tc.valid {
			t.Fatalf("valid=%v: ok=%v error=%d token=%d", tc.valid, checked.Ok, checked.Error, checked.ErrorToken)
		}
	}
}
