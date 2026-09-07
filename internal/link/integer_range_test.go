package link

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestNamedIntegerRangeConversion(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.22\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\ntype Count int\nfunc main(){for range Count(3){}}\n")},
	})
	program := &built.Units[built.Root].Program
	for i := 0; i < len(program.Tokens); i++ {
		if functionValueTokenEquals(program, i, "range") {
			end := concurrencyTopLevelToken(program, i+1, len(program.Tokens), "{")
			typ := ordinaryBuiltinExprType(program, i, i+1, end)
			if typ != "Count" || ordinaryUnderlyingType(program, typ, 0) != "int" {
				t.Fatalf("type=%q underlying=%q declared=%v local=%q", typ, ordinaryUnderlyingType(program, typ, 0), functionValueDeclaredType(program, "Count"), functionValueEnclosingLocalType(program, i, "Count"))
			}
		}
	}
	if !lowerIntegerRangesCore(program, false) {
		t.Fatal("integer range lowering failed")
	}
	for i := range program.Tokens {
		if functionValueTokenEquals(program, i, "range") {
			t.Fatal("integer range was not lowered")
		}
	}
}

func TestNamedIntegerConversionShadow(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.22\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\ntype Count int\nfunc main(){Count:=func(n int)[]int{return []int{n}};for range Count(3){}}\n")},
	})
	program := &built.Units[built.Root].Program
	for i := range program.Tokens {
		if functionValueTokenEquals(program, i, "range") {
			end := concurrencyTopLevelToken(program, i+1, len(program.Tokens), "{")
			if typ := ordinaryBuiltinExprType(program, i, i+1, end); typ == "Count" {
				t.Fatal("shadowed function mistaken for named type conversion")
			}
			return
		}
	}
	t.Fatal("missing range expression")
}
