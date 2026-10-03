package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestTypeSetInterfacesWithoutGenericDeclarations(t *testing.T) {
	for _, test := range []struct {
		name, source string
		valid        bool
	}{
		{"unused_union", `type C interface{~int|~string};func main(){}`, true},
		{"overlap", `type C interface{int|~int};func main(){}`, false},
		{"named_overlap", `type I int;type C interface{I|~int};func main(){}`, false},
		{"invalid_approximation", `type I int;type C interface{~I};func main(){}`, false},
		{"named_term", `type I int;type C interface{I};func main(){}`, true},
		{"parameter", `type C interface{~int};func F(v C){};func main(){}`, false},
		{"result", `type C interface{~int};func F()C{panic(1)};func main(){}`, false},
		{"variable", `type C interface{~int};var v C;func main(){}`, false},
		{"local_variable", `type C interface{~int};func main(){var v C;_=v}`, false},
		{"field", `type C interface{~int};type S struct{V C};func main(){}`, false},
		{"embedded_basic", `type A interface{M()};type B interface{A};func F(v B){};func main(){}`, true},
		{"embedded_variadic", `type A interface{M()};type B interface{A};func F(v ...B){};func main(){}`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;" + test.source)}})
			got := PrepareGenerics(graph)
			if got.Ok != test.valid {
				t.Fatalf("ok=%v want=%v: %s", got.Ok, test.valid, got.Message)
			}
		})
	}
}

func TestInterfaceTypeElementsLanguageVersion(t *testing.T) {
	for _, source := range []string{`type C interface{~int}`, `type C interface{int}`, `type I int;type C interface{I}`, `type C interface{comparable}`, `type A interface{};type C interface{A|A}`} {
		for _, version := range []string{"1.17", "1.18"} {
			graph := genericVersionTestGraph(t, version, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;" + source + ";func main(){}")}})
			if got := PrepareGenerics(graph); got.Ok != (version == "1.18") {
				t.Errorf("%s, %s: ok=%v: %s", source, version, got.Ok, got.Message)
			}
		}
	}
}
