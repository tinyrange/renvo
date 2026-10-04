package link

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"renvo.dev/internal/unit"
)

func TestImportRemovalUsesDeclarationBoundaries(t *testing.T) {
	for _, source := range []string{
		`package main;import "dep";func main(){}`,
		`package main;import (_ "dep";other "dep2");func main(){_ = other.Value}`,
		"package main\nimport (\n\"dep\"\nother \"dep2\"\n)\nfunc main(){_ = other.Value}\n",
		"package main\nimport \"dep\"; import other \"dep2\";func main(){_ = other.Value}\n",
	} {
		t.Run(source, func(t *testing.T) {
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, "case.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			var starts, ends []int
			for _, decl := range file.Decls {
				if decl, ok := decl.(*ast.GenDecl); ok && decl.Tok == token.IMPORT {
					start, end := set.Position(decl.Pos()).Offset, set.Position(decl.End()).Offset
					if end < len(source) && source[end] == ';' {
						end++
					}
					starts, ends = append(starts, start), append(ends, end)
				}
			}
			program := unit.Program{Package: "main"}
			if !reparseFunctionValueProgram(&program, []byte(source), nil, len(source), -1) {
				t.Fatal("parse unit")
			}
			actions := make([]tokenAction, len(program.Tokens))
			for _, imp := range file.Imports {
				path := coreTokenIndexByTextSpan(program.Tokens, set.Position(imp.Path.Pos()).Offset, set.Position(imp.Path.End()).Offset)
				if path < 0 {
					t.Fatal("import path token missing")
				}
				markCoreImportDeclTokens(&program, actions, unit.Import{NameTok: -1, PathTok: path})
			}
			for i, tok := range program.Tokens {
				want := false
				for j := range starts {
					want = want || tok.Start >= starts[j] && tok.Start < ends[j]
				}
				if got := actions[i] < 0; got != want {
					t.Fatalf("token %s at %d removed=%v, want %v", functionValueTokenText(&program, i), tok.Start, got, want)
				}
			}
		})
	}
}
