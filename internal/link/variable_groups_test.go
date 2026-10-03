package link

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"renvo.dev/internal/unit"
)

func TestVariableGroupsPreserveGoDeclarations(t *testing.T) {
	for _, source := range []string{
		`package main;var(a=17;b=19);func main(){print(a+b)}`,
		`package main;func main(){var(a=17;b=19);print(a+b)}`,
		`package main;func main(){var(a,b=17,19);print(a+b)}`,
		`package main;func main(){var(a=func()int{var(n=17;m=19);return n+m};b=23);print(a()+b)}`,
		"package main\nvar(\n a=func()int{\n return 17\n}\n b=\n19\n)\nfunc main(){print(a()+b)}",
		`package main;var();func main(){var();print(17)}`,
		`package main;func assembly()int;var(a=17;b=19);func main(){print(a+b)}`,
		`package main;var(a=17;b=19);func assembly()int;func main(){print(a+b)}`,
		`package main;func assembly()int;func main(){var(a=17;b=19);print(a+b)}`,
	} {
		t.Run(source, func(t *testing.T) {
			for _, transient := range []bool{false, true} {
				program := unit.Program{Package: "main"}
				if !reparseFunctionValueProgram(&program, []byte(source), nil, len(source), -1) {
					t.Fatal("parse source")
				}
				if !lowerVariableGroups(&program, transient) {
					t.Fatalf("lower grouped declarations: %s", program.Text)
				}

				for _, fn := range program.Funcs {
					if string(program.Text[fn.NameStart:fn.NameEnd]) == "assembly" {
						if fn.BodyStart != fn.EndTok || fn.BodyEnd != fn.EndTok {
							t.Fatalf("bodyless declaration lost its empty body: %+v", fn)
						}
					}
				}
				set := token.NewFileSet()
				file, err := parser.ParseFile(set, "main.go", program.Text, 0)
				if err != nil {
					t.Fatalf("parse output: %v\n%s", err, program.Text)
				}
				config := types.Config{}
				if _, err := config.Check("example.com/case", set, []*ast.File{file}, nil); err != nil {
					t.Fatalf("check output: %v\n%s", err, program.Text)
				}
			}
		})
	}
}
