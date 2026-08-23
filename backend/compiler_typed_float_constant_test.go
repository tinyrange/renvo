package main

import "testing"

func TestTypedFloatConstantDoesNotUseIntegerRepresentation(t *testing.T) {
	program := renvoParseProgram([]byte("package main\ntype Scalar = float64\nconst rowHeight Scalar = 34\n"))
	if !program.ok {
		t.Fatal("failed to parse source")
	}
	meta := renvoBuildMeta(&program)
	if !meta.ok {
		t.Fatal("failed to build metadata")
	}
	for i := 0; i < len(meta.globals); i++ {
		global := &meta.globals[i]
		if global.kind == renvoTokConst && renvoBytesEqualText(program.src, global.nameStart, global.nameEnd, "rowHeight") {
			if global.constValueOK == 0 {
				t.Fatal("rowHeight was not evaluated")
			}
			var gen renvoLinearGen
			gen.prog = &program
			gen.meta = &meta
			result := renvoEvalConstByName(&gen, global.nameStart, global.nameEnd)
			if result.ok {
				t.Fatalf("typed floating-point constant leaked through the integer constant evaluator as %d", result.value)
			}
			return
		}
	}
	t.Fatal("rowHeight constant not found")
}
