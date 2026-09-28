package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestImportedCollectionLiteralPrivacy(t *testing.T) {
	for _, tc := range []struct {
		expr  string
		valid bool
	}{
		{"[]v.Value{v.New(2)}", true},
		{"[1]v.Value{v.New(2)}", true},
		{"map[int]v.Value{1: v.New(2)}", true},
		{"[]*v.Value{nil}", true},
		{"[][]v.Value{{v.New(2)}}", true},
		{"[]v.Value{{}}", true},
		{"[]v.Value{{2}}", false},
		{"[]v.Value{{n: 2}}", false},
		{"map[int]v.Value{1: {n: 2}}", false},
		{"[][]v.Value{{{n: 2}}}", false},
		{"v.Value{2}", false},
		{"v.Value{n: 2}", false},
		{"[]v.Value{v.Value{n: 2}}", false},
		{"[]*v.Value{&v.Value{n: 2}}", false},
	} {
		graph := checkTestGraph(t, []load.SourceFile{
			{Path: "/repo/case/value/value.go", Src: []byte("package value\ntype Value struct { n int }\nfunc New(n int) Value { return Value{n} }\n")},
			{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nimport v \"example.com/case/value\"\nfunc main(){ _ = " + tc.expr + " }\n")},
		})
		result := CheckGraphCore(graph)
		if result.Ok != tc.valid {
			t.Errorf("%s: ok=%v error=%d", tc.expr, result.Ok, result.Error)
		}
	}
}

func TestImportedSymbolVisibility(t *testing.T) {
	for _, tc := range []struct {
		name  string
		valid bool
	}{
		{"Value", true}, {"value", false}, {"É", true}, {"é", false},
	} {
		graph := checkTestGraph(t, []load.SourceFile{
			{Path: "/repo/case/value/value.go", Src: []byte("package value\nvar Value, value, É, é int\n")},
			{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nimport v \"example.com/case/value\"\nfunc main(){_=v." + tc.name + "}\n")},
		})
		result := CheckGraphCore(graph)
		if result.Ok != tc.valid {
			t.Errorf("%s: ok=%v error=%d", tc.name, result.Ok, result.Error)
		}
	}
}

func TestAuditImportedVisibility(t *testing.T) {
	for _, expression := range []string{"lib.hidden", "lib.S{hidden: 1}", "lib.S{1, 2}"} {
		graph := checkTestGraph(t, []load.SourceFile{
			{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nimport \"example.com/case/lib\"\nfunc main() { _ = " + expression + " }")},
			{Path: "/repo/case/lib/lib.go", Src: []byte("package lib\nvar hidden = 2\ntype S struct { hidden int; X int }")},
		})
		result := CheckGraphCore(graph)
		if result.Ok || result.Error != CheckErrUndefined {
			t.Fatalf("%s: accepted or wrong error %d", expression, result.Error)
		}
	}
}

func TestUnicodeImportedVisibility(t *testing.T) {
	for _, name := range []string{"Ω", "π", "世界"} {
		graph := checkTestGraph(t, []load.SourceFile{
			{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nimport \"example.com/case/lib\"\nfunc main(){_=lib." + name + "()}")},
			{Path: "/repo/case/lib/lib.go", Src: []byte("package lib\nfunc Ω() int{return 1};func π() int{return 2};func 世界() int{return 3}")},
		})
		result := CheckGraphCore(graph)
		if result.Ok != (name == "Ω") {
			t.Fatalf("%s: ok=%v error=%d", name, result.Ok, result.Error)
		}
	}
}
