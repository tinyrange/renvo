package link

import (
	"reflect"
	"renvo.dev/internal/unit"
	"strings"
	"testing"
)

func TestBuiltinCallEditsMatchParsedTables(t *testing.T) {
	for _, source := range []string{
		"package main\nfunc value(a,b int) int { return min(max(a,b),min(a,3)) }\nfunc main(){println(value(4,7))}\n",
		"package main\ntype T struct{x int}\nfunc (v T) value(r rune) string { return string(r) }\nfunc main(){v:=T{x:3};println(v.value('雪'))}\n",
		"package main\nvar n = min(3,4)\nfunc main(){a:=[]int{1,2};clear(a);println(n)}\n",
		"package main\nfunc main(){a:=[]int{1};clear(a);n:=min(3,4);clear(a);println(n,string(65))}\n",
		"package main\nfunc main(){a:=min(\n3, /* comment\non another line */ 4);println(a,string(\n65))}\n\n",
		"package main\nfunc main(){println(string(len(`" + strings.Repeat("\n", 65536) + "`)))}\n",
	} {
		for _, transient := range []bool{false, true} {
			var program unit.Program
			if !reparseFunctionValueProgram(&program, []byte(source), nil, len(source), -1) {
				t.Fatal("parse failed", source)
			}
			if transient {
				tokens := make([]unit.Token, len(program.Tokens), len(program.Tokens)+1024)
				copy(tokens, program.Tokens)
				program.Tokens = tokens
				text := make([]byte, len(program.Text), len(program.Text)+4096)
				copy(text, program.Text)
				program.Text = text
			}
			before := &program.Tokens[0]
			textBefore := &program.Text[0]
			if !lowerOrdinaryBuiltins(&program, transient) {
				t.Fatal("rewrite failed", source)
			}
			if transient && before != &program.Tokens[0] {
				t.Fatal("transient rewrite replaced the reserved token table")
			}
			if transient && textBefore != &program.Text[0] {
				t.Fatal("transient rewrite replaced the reserved source buffer")
			}
			var parsed unit.Program
			if !reparseFunctionValueProgram(&parsed, program.Text, nil, len(program.Text), -1) {
				t.Fatal("parse failed", string(program.Text))
			}
			if !reflect.DeepEqual(program.Tokens, parsed.Tokens) || !reflect.DeepEqual(program.Decls, parsed.Decls) || !reflect.DeepEqual(program.Funcs, parsed.Funcs) {
				t.Fatalf("rewritten metadata differs from parser for %s", source)
			}
		}
	}
}

func TestMaxOnlyProgramNeedsBuiltinLowering(t *testing.T) {
	const source = "package main\nfunc main(){a:=3;b:=4;println(max(a,b))}\n"
	var program unit.Program
	if !reparseFunctionValueProgram(&program, []byte(source), nil, len(source), -1) {
		t.Fatal("parse failed")
	}
	_, _, builtins := functionValueProgramNeedsLowering(&program)
	if !builtins {
		t.Fatal("max-only program did not request builtin lowering")
	}
}
