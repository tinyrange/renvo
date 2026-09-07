package link

import (
	"testing"

	"renvo.dev/internal/unit"
)

func TestCallbackTupleResultInference(t *testing.T) {
	source := []byte(`package main
type Loader func(string) (map[string]int, error)
type Module struct { loader Loader }
type Compiler struct { module *Module }
func (c *Compiler) run() {
 values, err := c.module.loader("answer")
 _ = err
 _ = values
}
`)
	program := unit.Program{Package: "main"}
	if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
		t.Fatal("parse")
	}
	for tok := 0; tok < len(program.Tokens); tok++ {
		if functionValueTokenEquals(&program, tok, "values") && functionValueTokenEquals(&program, tok+1, ",") {
			if got := functionValueLocalCallResultType(&program, tok); functionValueCompactTypeText(got) != "map[string]int" {
				t.Fatalf("result type = %q; callee type = %q", got, ordinaryBuiltinExprType(&program, tok+4, tok+4, tok+9))
			}
			return
		}
	}
	t.Fatal("binding not found")
}

func TestCallbackUnnamedCompositeSignature(t *testing.T) {
	for _, typ := range []string{"map[string]int", "chan int", "[]int", "struct{ N int }"} {
		source := []byte("package main\ntype F func(" + typ + ") (" + typ + ", error)\n")
		program := unit.Program{Package: "main"}
		if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
			t.Fatal("parse", typ)
		}
		found := false
		for tok := 0; tok < len(program.Tokens); tok++ {
			if !functionValueTokenEquals(&program, tok, "func") {
				continue
			}
			sig, _, ok := parseFunctionValueSignature(&program, tok, "F")
			if !ok || len(sig.paramTypes) != 1 || len(sig.resultTypes) != 2 || functionValueCompactTypeText(sig.paramTypes[0]) != functionValueCompactTypeText(typ) || functionValueCompactTypeText(sig.resultTypes[0]) != functionValueCompactTypeText(typ) {
				t.Fatalf("%s: params=%v results=%v", typ, sig.paramTypes, sig.resultTypes)
			}
			found = true
			break
		}
		if !found {
			t.Fatal("signature not found", typ)
		}
	}
}
