package link

import (
	"os"
	"renvo.dev/internal/build"
	"renvo.dev/internal/load"
	"strings"
	"testing"
)

func TestConstraintValueExpressionsLink(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"value_interface", `type Value[T any]=interface{Value()T};type Box[T any]struct{Item T};func(b Box[T])Value()T{return b.Item};func Read[T any](v Value[T])T{return v.Value()};func main(){_=Read(Value[int](Box[int]{42}))}`},
		{"local_function", `type C interface{~int};func main(){C,v:=func(v int)int{return v+1},40;_=C(v)}`},
		{"header_function", `type C interface{~int};func main(){if C:=func(v int)int{return v+2};C(40)!=42{panic(1)}}`},
		{"closure_parameter", `type C interface{~int};func main(){f:=func(C func(int)int)int{return C(41)};_=f(func(v int)int{return v+1})}`},
		{"local_alias", `type C interface{~int};func main(){type C=int;_=C(42)}`},
		{"global_function", `var comparable=func(v int)int{return v+1};func main(){_=comparable(41)}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := buildFromFiles(t, []load.SourceFile{
				{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.25\n")},
				{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;" + tc.source)},
			})
			if got := LinkBuildCore(input); !got.Ok {
				t.Fatalf("link: %d", got.Error)
			}
		})
	}
}

func TestConstraintValueExpressionsAcrossPackages(t *testing.T) {
	var files []load.SourceFile
	files = append(files, load.SourceFile{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.25\n")})
	for _, path := range []string{"cmd/app/main.go", "model/model.go"} {
		src, err := os.ReadFile("../../frontend_tests/regressions/generic_constraint_values/" + path)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, load.SourceFile{Path: "/repo/case/" + path, Src: []byte(strings.ReplaceAll(string(src), "example.com/genericconstraintvalues", "example.com/case"))})
	}
	input := buildFromFiles(t, files)
	if linked := LinkBuildCore(input); !linked.Ok {
		t.Fatalf("link %d", linked.Error)
	}
}

func TestFunctionValueTypedLiteralInitializer(t *testing.T) {
	workspace := load.LoadWorkspace("/repo/case", "/std", "./cmd/app", []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.25\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
func Id[T any](v T)T{return v}
func main(){
 var callback func(int)int=func(v int)int{return v+1}
 print(callback(41))
}`)},
	})
	input := build.BuildPrograms(workspace.Graph)
	if !input.Ok {
		t.Fatalf("build: %d detail %d token %d: %s", input.Error, input.ErrorDetail, input.ErrorToken, input.ErrorMessage)
	}
	linked := LinkBuildCore(input)
	if !linked.Ok {
		t.Fatalf("link %d", linked.Error)
	}
	if strings.Contains(string(linked.Program.Text), "= func(") || strings.Contains(string(linked.Program.Text), "=func(") {
		t.Fatalf("literal initializer reached backend:\n%s", linked.Program.Text)
	}
}

func TestFunctionValueInitializerUsesOuterBinding(t *testing.T) {
	input := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.25\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main;type Callback func(int)int;func main(){int:=int(42);print(int)}`)},
	})
	linked := LinkBuildCore(input)
	if !linked.Ok {
		t.Fatalf("link %d", linked.Error)
	}
	if !strings.Contains(string(linked.Program.Text), "int:=int(42)") {
		t.Fatalf("initializer did not retain the outer conversion:\n%s", linked.Program.Text)
	}
}

func TestGenericPredeclaredShadowsLink(t *testing.T) {
	var files []load.SourceFile
	files = append(files, load.SourceFile{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.25\n")})
	for _, path := range []string{"cmd/app/main.go", "model/model.go"} {
		src, err := os.ReadFile("../../frontend_tests/regressions/generic_predeclared_shadow/" + path)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, load.SourceFile{Path: "/repo/case/" + path, Src: []byte(strings.ReplaceAll(string(src), "example.com/genericpredeclaredshadow", "example.com/case"))})
	}
	linked := LinkBuildCore(buildFromFiles(t, files))
	if !linked.Ok {
		t.Fatalf("link %d", linked.Error)
	}
}
