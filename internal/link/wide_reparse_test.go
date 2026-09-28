package link

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/unit"
	"strings"
	"testing"
)

func TestBuiltinReparseAcrossCombinedSourceLineLimit(t *testing.T) {
	padding := strings.Repeat("\n", 34000)
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/convert/convert.go", Src: []byte("package convert\n" + padding + "func Text(r rune) string { return string(r) }\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nimport \"example.com/case/convert\"\n" + padding + "func main() { if convert.Text(65) != \"A\" { panic(\"conversion\") }; print(\"PASS\\n\") }\n")},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("wide builtin reparse failed: %d", linked.Error)
	}
}

func TestReparseKeepsStandaloneDeclarationKeywords(t *testing.T) {
	source := []byte("package main\nconst size = 32\nvar buffer [size]byte\ntype Record struct { Value int }\nfunc main() {}\n")
	program := unit.Program{Package: "main"}
	if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
		t.Fatal("reparse")
	}
	for _, decl := range program.Decls {
		if program.Tokens[decl.StartTok].KindLine&255 != decl.Kind {
			t.Fatal("reparse dropped declaration keyword")
		}
	}
}
