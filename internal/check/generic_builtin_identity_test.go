package check

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"renvo.dev/internal/load"
	"strings"
	"testing"
)

// Check the concrete declarations with Go's type checker as well as Renvo's:
// accepting the spelling of a different named type is not sufficient.
type genericConcreteImporter struct {
	t        *testing.T
	graph    load.Graph
	packages map[string]*types.Package
}

func (i *genericConcreteImporter) Import(path string) (*types.Package, error) {
	if pkg := i.packages[path]; pkg != nil {
		return pkg, nil
	}
	for _, pkg := range i.graph.Packages {
		if pkg.Ref.ImportPath != path {
			continue
		}
		fset := token.NewFileSet()
		var files []*ast.File
		for _, source := range pkg.Files {
			file, err := parser.ParseFile(fset, source.Path, source.Src, 0)
			if err != nil {
				i.t.Fatal(err)
			}
			files = append(files, file)
		}
		config := types.Config{Importer: i}
		checked, err := config.Check(path, fset, files, nil)
		if err != nil {
			i.t.Fatal(err)
		}
		i.packages[path] = checked
		return checked, nil
	}
	i.t.Fatalf("missing concrete import %q", path)
	return nil, nil
}

func TestGenericInferredBuiltinIdentityWithPackageShadow(t *testing.T) {
	for _, name := range []string{"bool", "string", "int", "uint", "uintptr", "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "complex64", "complex128", "error", "byte", "rune", "any"} {
		t.Run(name, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{
				{Path: "/repo/case/lib/value.go", Src: []byte("package lib;func Value()(v " + name + "){return}")},
				{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nimport \"example.com/case/lib\"\ntype " + name + " struct{}\nfunc Id[T interface{}](v T)T{return v}\nfunc main(){_=Id(lib.Value())}")},
			})
			prepared := PrepareGenerics(graph)
			if !prepared.Ok {
				t.Fatal(prepared.Message)
			}
			if checked := CheckGraphCore(prepared.Graph); !checked.Ok {
				t.Fatalf("concrete check: %+v", checked)
			}
			importer := genericConcreteImporter{t: t, graph: prepared.Graph, packages: map[string]*types.Package{}}
			pkg, _ := importer.Import(graph.Root)
			found := false
			for _, symbol := range pkg.Scope().Names() {
				if !strings.HasPrefix(symbol, "RenvoGenericInstance_") {
					continue
				}
				fn, ok := pkg.Scope().Lookup(symbol).(*types.Func)
				if !ok {
					continue
				}
				got := fn.Type().(*types.Signature).Results().At(0).Type()
				want := types.Universe.Lookup(name).Type()
				if !types.Identical(got, want) {
					t.Fatalf("inferred %s instead of universe %s", got, name)
				}
				found = true
			}
			if !found {
				t.Fatal("no concrete function")
			}
			if prepared.Graph.Packages[checkRootPackage(t, graph)].Ref.ImportPath != graph.Root {
				t.Fatal("original package indices changed")
			}
			if again := PrepareGenerics(prepared.Graph); !again.Ok || len(again.Graph.Packages) != len(prepared.Graph.Packages) {
				t.Fatal("preparation is not idempotent")
			}
		})
	}
}

func TestGenericInferredLiteralIdentityWithPackageShadow(t *testing.T) {
	for _, tc := range []struct{ name, value string }{{"bool", "true"}, {"string", `"value"`}, {"int", "42"}, {"int32", "'x'"}, {"float64", "1.5"}, {"complex128", "2i"}} {
		t.Run(tc.name, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\ntype " + tc.name + " struct{}\nfunc Id[T interface{}](v T)T{return v}\nfunc main(){_=Id(" + tc.value + ")}")}})
			prepared := PrepareGenerics(graph)
			if !prepared.Ok {
				t.Fatal(prepared.Message)
			}
			importer := genericConcreteImporter{t: t, graph: prepared.Graph, packages: map[string]*types.Package{}}
			importer.Import(graph.Root)
		})
	}
}

func TestGenericBuiltinIdentityWithValueAndImportShadows(t *testing.T) {
	for _, declaration := range []string{"var int=0", "const int=0", "func int(){}", "type int=string", "import int \"example.com/case/lib\""} {
		t.Run(declaration, func(t *testing.T) {
			use := ""
			if strings.HasPrefix(declaration, "import") {
				use = ";_=int.Value()"
			}
			graph := genericTestGraph(t, []load.SourceFile{
				{Path: "/repo/case/lib/value.go", Src: []byte("package lib;func Value()int{return 42}")},
				{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + declaration + "\nfunc Id[T any](v T)T{return v}\nfunc main(){_=Id(42)" + use + "}")},
			})
			prepared := PrepareGenerics(graph)
			if !prepared.Ok {
				t.Fatal(prepared.Message)
			}
			importer := genericConcreteImporter{t: t, graph: prepared.Graph, packages: map[string]*types.Package{}}
			importer.Import(graph.Root)
		})
	}
}

func TestGenericBuiltinBridgeAvoidsUserNamesAndIsDeterministic(t *testing.T) {
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type int string
var __renvo_generic_package_2 = 0
func Id[T any](v T)T{return v}
func main(){_=Id(42)}
`)}})
	// Occupying the default generated package path must not hijack aliases.
	graph.Packages = append(graph.Packages, load.Package{Ref: load.PackageRef{Kind: load.PackageInModule, ImportPath: "renvo.generated/generic/builtins", Ok: true}, Name: "occupied", Ok: true})
	first, second := PrepareGenerics(graph), PrepareGenerics(graph)
	if !first.Ok || !second.Ok {
		t.Fatalf("prepare: %s / %s", first.Message, second.Message)
	}
	if len(first.Graph.Packages) != len(graph.Packages)+1 {
		t.Fatal("missing alias package")
	}
	for p := range first.Graph.Packages {
		if first.Graph.Packages[p].Ref.ImportPath != second.Graph.Packages[p].Ref.ImportPath {
			t.Fatal("unstable import path")
		}
		for f := range first.Graph.Packages[p].Files {
			if string(first.Graph.Packages[p].Files[f].Src) != string(second.Graph.Packages[p].Files[f].Src) {
				t.Fatal("unstable concrete source")
			}
		}
	}
	if first.Graph.Packages[len(graph.Packages)].Ref.ImportPath == graph.Packages[len(graph.Packages)-1].Ref.ImportPath {
		t.Fatal("user package path reused")
	}
	importer := genericConcreteImporter{t: t, graph: first.Graph, packages: map[string]*types.Package{}}
	importer.Import(graph.Root)
	if strings.Contains(string(graph.Packages[0].Files[0].Src), "RenvoGenericInstance") {
		t.Fatal("original source mutated")
	}
}

func TestGenericUnshadowedBuiltinsNeedNoBridgePackage(t *testing.T) {
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc Id[T any](v T)T{return v}\nfunc main(){_=Id(42)}")}})
	prepared := PrepareGenerics(graph)
	if !prepared.Ok || len(prepared.Graph.Packages) != len(graph.Packages) {
		t.Fatalf("unnecessary builtin package: %s", prepared.Message)
	}
}
