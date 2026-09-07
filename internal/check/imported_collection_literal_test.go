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
