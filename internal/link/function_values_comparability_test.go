package link

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	wireunit "renvo.dev/backend/unit"
	"renvo.dev/internal/load"
)

func TestLoweredFunctionValuesRetainComparability(t *testing.T) {
	input := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.25\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type Callback func()
type Alias = Callback
type AnonymousCallback = func()
type SecondAnonymous = func()
type Empty [0]Callback
type Container struct{Value Callback}
type Anonymous struct{Value func()}
type Pointer *Callback
type Ordinary struct{Kind int;Value *int}
func target(){}
func main(){var callback Callback=target;callback()}
`)},
	})
	for _, mode := range []string{"persistent", "incremental", "transient", "incremental_transient"} {
		var linked Result
		switch mode {
		case "incremental":
			linked = LinkBuildCoreIncremental(input)
		case "transient":
			linked = LinkBuildCoreTransient(input)
		case "incremental_transient":
			session := BeginPackageSession(input, true)
			for !session.Step() {
			}
			linked = session.Result()
		default:
			linked = LinkBuildCore(input)
		}
		if !linked.Ok {
			t.Fatalf("mode=%s link error=%d", mode, linked.Error)
		}
		program, err := wireunit.Unmarshal(linked.Data)
		if err != nil {
			t.Fatal(err)
		}
		set := token.NewFileSet()
		file, err := parser.ParseFile(set, "linked.go", program.Text, 0)
		if err != nil {
			t.Fatal(err)
		}
		// Check the actual linked type declarations independently of generated
		// runtime bodies. Comparability must follow from their Go type semantics.
		var declarations []ast.Decl
		for _, declaration := range file.Decls {
			if gen, ok := declaration.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
				declarations = append(declarations, gen)
			}
		}
		file.Decls = declarations
		config := types.Config{}
		pkg, err := config.Check("example.com/case", set, []*ast.File{file}, nil)
		if err != nil {
			t.Fatal(err)
		}
		callback := pkg.Scope().Lookup("Callback").Type()
		anonymous := pkg.Scope().Lookup("AnonymousCallback").Type()
		second := pkg.Scope().Lookup("SecondAnonymous").Type()
		if types.Identical(callback, anonymous) || !types.Identical(anonymous, second) || !types.Identical(callback.Underlying(), anonymous.Underlying()) {
			t.Fatalf("mode=%s function identities or shared storage lost: defined=%v anonymous=%v second=%v", mode, callback, anonymous, second)
		}
		for _, tc := range []struct {
			name       string
			comparable bool
		}{
			{"Callback", false}, {"Alias", false}, {"AnonymousCallback", false}, {"SecondAnonymous", false}, {"Empty", false},
			{"Container", false}, {"Anonymous", false}, {"Pointer", true}, {"Ordinary", true},
		} {
			object := pkg.Scope().Lookup(tc.name)
			if object == nil || types.Comparable(object.Type()) != tc.comparable {
				t.Fatalf("mode=%s %s: type=%v want comparable=%v", mode, tc.name, object, tc.comparable)
			}
		}
	}
}
