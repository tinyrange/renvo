package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func genericTestGraph(t *testing.T, files []load.SourceFile) load.Graph {
	t.Helper()
	return genericVersionTestGraph(t, "1.25.5", files)
}

func genericVersionTestGraph(t *testing.T, version string, files []load.SourceFile) load.Graph {
	t.Helper()
	module := load.Module{Root: "/repo/case", Path: "example.com/case", GoVersion: version, Ok: true}
	graph := load.LoadGraph(module, "/std", "/repo/case", "./cmd/app", files)
	if !graph.Ok {
		t.Fatalf("load failed: error=%d package=%d", graph.Error, graph.ErrorPackage)
	}
	return graph
}

func TestGenericLanguageVersions(t *testing.T) {
	for _, test := range []struct {
		name, source, before, since string
	}{
		{"function", `func F[T any](v T)T{return v};func main(){}`, "1.17", "1.18"},
		{"type", `type Box[T any] struct{Value T};func main(){}`, "1.17", "1.18"},
		{"alias", `type Box[T any] = []T;func main(){}`, "1.23", "1.24"},
		{"comparable_function", `func F[T comparable](){};func main(){F[any]()}`, "1.19", "1.20"},
		{"comparable_struct", `type Box[T comparable] struct{Value T};func main(){var x Box[struct{Value any}];_=x}`, "1.19", "1.20"},
		{"function_value", `func Id[T any](v T)T{return v};func main(){var f func(int)int=Id;_=f}`, "1.20", "1.21"},
		{"partial_value", `func F[A,B any](v A,w B)B{return w};func main(){var f func(int,string)string=F[int];_=f}`, "1.20", "1.21"},
		{"argument_value", `func Id[T any](v T)T{return v};func Apply[T any](f func(T)T,v T)T{return f(v)};func main(){_=Apply(Id,1)}`, "1.20", "1.21"},
		{"interface_inference", `type V int;func(v V)Value()int{return int(v)};func F[T any](v interface{Value()T})T{return v.Value()};func main(){_=F(V(1))}`, "1.20", "1.21"},
		{"min", `func F[T ~int](a,b T)T{return min(a,b)};func main(){_=F(1,2)}`, "1.20", "1.21"},
		{"clear", `func F[T any](s []T){clear(s)};func main(){F([]int{})}`, "1.20", "1.21"},
	} {
		for _, version := range []string{test.before, test.since} {
			t.Run(test.name+"/"+version, func(t *testing.T) {
				graph := genericVersionTestGraph(t, version, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;" + test.source)}})
				got := PrepareGenerics(graph)
				if got.Ok != (version == test.since) {
					t.Fatalf("ok=%v: %s", got.Ok, got.Message)
				}
			})
		}
	}
}

func TestGenericVersionUsesInstantiationFile(t *testing.T) {
	for _, version := range []string{"1.17", "1.18"} {
		graph := genericVersionTestGraph(t, version, []load.SourceFile{
			{Path: "/repo/case/cmd/app/definition.go", Src: []byte("//go:build go1.21\n\npackage main;type Box[T any] struct{Value T}")},
			{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;func(b Box[T])Get()T{return b.Value};func main(){}")},
		})
		if got := PrepareGenerics(graph); got.Ok != (version == "1.18") {
			t.Fatalf("receiver file %s: ok=%v: %s", version, got.Ok, got.Message)
		}
	}
	for _, version := range []string{"1.19", "1.20"} {
		graph := genericVersionTestGraph(t, version, []load.SourceFile{
			{Path: "/repo/case/cmd/app/definition.go", Src: []byte("//go:build go1.21\n\npackage main;type Box[T comparable] struct{Value T};func F[T comparable](){}")},
			{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;func main(){F[any]();var b Box[any];_=b}")},
		})
		if got := PrepareGenerics(graph); got.Ok != (version == "1.20") {
			t.Fatalf("%s: ok=%v: %s", version, got.Ok, got.Message)
		}
	}
	graph := genericVersionTestGraph(t, "1.23", []load.SourceFile{
		{Path: "/repo/case/cmd/app/alias.go", Src: []byte("//go:build go1.24\n\npackage main;type Box[T any] = []T")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;func main(){var b Box[int];_=b}")},
	})
	if got := PrepareGenerics(graph); !got.Ok {
		t.Fatalf("alias declaration owns its language version: %s", got.Message)
	}
}

func TestGenericExplicitFunctionValueBeforeInferenceVersion(t *testing.T) {
	graph := genericVersionTestGraph(t, "1.18", []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;func Id[T any](v T)T{return v};func main(){var f func(int)int=Id[int];_=f}")}})
	if got := PrepareGenerics(graph); !got.Ok {
		t.Fatal(got.Message)
	}
}

func TestGenericLegacyInterfaceArgumentAssignment(t *testing.T) {
	for _, source := range []string{
		`type V int;func(v V)Value()int{return int(v)};func F[T any](v interface{Value()T},value T)T{return value};func main(){_=F(V(1),1)}`,
		`type V int;func(v V)Value()int{return int(v)};func F[T any](v interface{Value()T})T{return v.Value()};func main(){_=F[int](V(1))}`,
	} {
		graph := genericVersionTestGraph(t, "1.18", []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;" + source)}})
		if got := PrepareGenerics(graph); !got.Ok {
			t.Fatal(got.Message)
		}
	}
}
