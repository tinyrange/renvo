package link

import (
	"reflect"
	"renvo.dev/internal/unit"
	"testing"
)

func TestBuiltinCallEditsMatchParsedTables(t *testing.T) {
	for _, source := range []string{
		"package main\nfunc value(a,b int) int { return min(max(a,b),min(a,3)) }\nfunc main(){println(value(4,7))}\n",
		"package main\ntype T struct{x int}\nfunc (v T) value(r rune) string { return string(r) }\nfunc main(){v:=T{x:3};println(v.value('雪'))}\n",
		"package main\nvar n = min(3,4)\nfunc main(){a:=[]int{1,2};clear(a);println(n)}\n",
	} {
		var program unit.Program
		if !reparseFunctionValueProgram(&program, []byte(source), nil, len(source), -1) || !lowerOrdinaryBuiltins(&program, false) {
			t.Fatal("rewrite failed", source)
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
