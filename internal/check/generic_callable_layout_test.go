package check

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"renvo.dev/internal/load"
)

func TestGenericCallableLayoutPreservesOperandUses(t *testing.T) {
	for _, body := range []string{
		`var fn func(T);var array [unsafe.Sizeof(fn)]byte;_=array`,
		`var fn func(T);const n=unsafe.Sizeof(fn);var array [n]byte;_=array`,
		`var fn func(T);var value any;_,_=value.([unsafe.Sizeof(fn)]byte)`,
		`const n=unsafe.Sizeof(Identity[T]);var array[n]byte;_=array`,
	} {
		source := []byte("package main;import \"unsafe\";func Identity[T any](v T)T{return v};func F[T any](){" + body + "};func main(){F[int]()}")
		checkGo := func(source []byte) {
			t.Helper()
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, "main.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			config := types.Config{Importer: importer.Default()}
			if _, err := config.Check("example.com/case", set, []*ast.File{file}, nil); err != nil {
				t.Fatalf("Go check: %v\n%s", err, source)
			}
		}
		checkGo(source)
		graph := genericTestGraph(t, []load.SourceFile{
			{Path: "/repo/case/cmd/app/main.go", Src: source},
			{Path: "/std/unsafe/unsafe.go", Src: []byte("package unsafe;type Pointer *byte")},
		})
		prepared := PrepareGenerics(graph)
		if !prepared.Ok {
			t.Fatal(prepared.Message)
		}
		if result := CheckGraphCore(prepared.Graph); !result.Ok {
			t.Fatalf("concrete checker: %+v", result)
		}
		checkGo(prepared.Graph.Packages[0].Files[0].Src)
	}
}
