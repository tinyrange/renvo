package link

import (
	"renvo.dev/internal/unit"
	"testing"
)

func TestIndexedLocalType(t *testing.T) {
	for _, tc := range []struct{ container, expr, want string }{
		{"string", "input[0]", "byte"},
		{"Text", "input[0]", "byte"},
		{"[]rune", "input[0]", "rune"},
		{"[2]int64", "input[0]", "int64"},
		{"string", "input[0:1]", "string"},
		{"[]rune", "input[0:1]", "[]rune"},
	} {
		source := []byte("package main\ntype Text string\nfunc check(input " + tc.container + ") {\n value := " + tc.expr + "\n _ = value\n}\n")
		program := unit.Program{Package: "main"}
		if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
			t.Fatal("parse")
		}
		found := false
		for tok := 0; tok < len(program.Tokens); tok++ {
			if functionValueTokenEquals(&program, tok, "value") && functionValueTokenEquals(&program, tok-1, "=") {
				found = true
				if got := functionValueEnclosingLocalType(&program, tok, "value"); got != tc.want {
					t.Errorf("%s %s: got %q want %q", tc.container, tc.expr, got, tc.want)
				}
			}
		}
		if !found {
			t.Fatal("reference not found")
		}
	}
}
