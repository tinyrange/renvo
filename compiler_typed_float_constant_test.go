package main

import "testing"

func TestTypedFloatConstantUsesFloatRepresentation(t *testing.T) {
	program := rtgParseProgram([]byte("package main\ntype Scalar = float64\nconst rowHeight Scalar = 34\n"))
	if !program.ok {
		t.Fatal("failed to parse source")
	}
	meta := rtgBuildMeta(&program)
	if !meta.ok {
		t.Fatal("failed to build metadata")
	}
	for i := 0; i < len(meta.globals); i++ {
		global := &meta.globals[i]
		if global.kind == rtgTokConst && rtgBytesEqualText(program.src, global.nameStart, global.nameEnd, "rowHeight") {
			if global.constValueOK == 0 {
				t.Fatal("rowHeight was not evaluated")
			}
			var gen rtgLinearGen
			gen.prog = &program
			gen.meta = &meta
			result := rtgEvalConstByName(&gen, global.nameStart, global.nameEnd)
			if !result.ok || result.value != 34*4 {
				t.Fatalf("rowHeight representation = %d (ok %v), want %d", result.value, result.ok, 34*4)
			}
			return
		}
	}
	t.Fatal("rowHeight constant not found")
}
