package link

import (
	"testing"

	"renvo.dev/internal/unit"
)

func TestMapLiteralTypeStartsPreserveOutermostType(t *testing.T) {
	source := []byte(`package main
type Entry struct { N int }
type Table map[string]int
func main() {
 a := []map[string]int{{"one": 1}, {"two": 2}}
 b := [2]map[string]int{{"one": 1}, {"two": 2}}
 c := map[string]map[string]int{"outer": {"inner": 3}}
 d := map[string]func() int{"call": func() int { return 4 }}
 e := []struct { N int }{{N: 5}}
 f := Table{"six": 6}
 g := &Entry{N: 7}
 for i := 0; i < 3; i++ { if i > 0 { _ = i } }
 _ = a; _ = b; _ = c; _ = d; _ = e; _ = f; _ = g
}
`)
	program := unit.Program{Package: "main"}
	if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
		t.Fatal("parse")
	}
	starts := mapLowerLiteralTypeStarts(&program)
	functionBodies := map[int]bool{}
	for _, fn := range program.Funcs {
		functionBodies[fn.BodyStart] = true
	}
	for tok := range program.Tokens {
		if !functionValueTokenEquals(&program, tok, "func") || !functionValueTokenEquals(&program, tok+1, "(") {
			continue
		}
		_, body, ok := parseFunctionValueSignature(&program, tok, "")
		if ok && functionValueTokenEquals(&program, body, "{") && !functionValueLiteralTypePosition(&program, tok, body) {
			functionBodies[body] = true
		}
	}
	braces := 0
	for open := range program.Tokens {
		if !functionValueTokenEquals(&program, open, "{") {
			continue
		}
		braces++
		// Independent reference: the previous exhaustive backwards search.
		want := -1
		for candidate := open - 1; candidate >= 0; candidate-- {
			if functionValueTypeEnd(&program, candidate) == open {
				want = candidate
			}
		}
		if want < 0 {
			want = functionValuePrimaryStart(&program, open-1)
		}
		// A result type before a function body is not a composite literal.
		if functionBodies[open] {
			want = -1
		}
		if starts[open] != want {
			t.Errorf("brace %d: got start %d, want %d", open, starts[open], want)
		}
	}
	if braces < 15 {
		t.Fatalf("insufficient literal/block coverage: %d braces", braces)
	}
}
