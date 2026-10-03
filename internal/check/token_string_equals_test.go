package check

import (
	"testing"

	"renvo.dev/internal/syntax"
)

func TestTokenStringEqualsPreservesEmptyAndInvalidSpans(t *testing.T) {
	file := syntax.File{Src: []byte{'a', 0, 255, 'b'}}
	for _, span := range [][2]int32{{0, 1}, {1, 2}, {2, 3}, {0, 4}, {4, 4}, {-1, 1}, {2, 1}, {3, 5}} {
		file.Tokens = []syntax.Token{{Start: span[0], End: span[1]}}
		for _, index := range []int{-1, 0, 1} {
			for _, want := range []string{"", "a", "b", "\x00", "\xff", "a\x00\xffb"} {
				if got := tokenStringEquals(&file, index, want); got != (tokenString(&file, index) == want) {
					t.Fatalf("span=%v index=%d want=%q: got %v", span, index, want, got)
				}
			}
		}
	}
}

func TestTokenKindMatchesParsedKeywords(t *testing.T) {
	source := []byte(`package import const var type func struct interface map return if else for range switch case default break continue goto defer go select chan fallthrough function "func"`)
	file := syntax.File{Src: source, Tokens: syntax.Scan(source)}
	keywords := map[string]int{"package": syntax.TokenPackage, "import": syntax.TokenImport, "const": syntax.TokenConst, "var": syntax.TokenVar, "type": syntax.TokenType, "func": syntax.TokenFunc, "struct": syntax.TokenStruct, "interface": syntax.TokenInterface, "map": syntax.TokenMap, "return": syntax.TokenReturn, "if": syntax.TokenIf, "else": syntax.TokenElse, "for": syntax.TokenFor, "range": syntax.TokenRange, "switch": syntax.TokenSwitch, "case": syntax.TokenCase, "default": syntax.TokenDefault, "break": syntax.TokenBreak, "continue": syntax.TokenContinue, "goto": syntax.TokenGoto, "defer": syntax.TokenDefer, "go": syntax.TokenGo, "select": syntax.TokenSelect, "chan": syntax.TokenChan, "fallthrough": syntax.TokenFallthrough}
	for index := -1; index <= len(file.Tokens); index++ {
		for word, kind := range keywords {
			if got, want := tokenKindIs(&file, index, kind), tokenTextIs(&file, index, word); got != want {
				t.Fatalf("token=%d keyword=%q: got %v, want %v", index, word, got, want)
			}
		}
	}
}

func TestOperatorCharMatchesParsedTokenText(t *testing.T) {
	source := []byte("package p; func f(){ var 字符 string; x := a+b-c*d/e%f; x &= 3; if x != 0 && y == 1 || z <= 2 { f(x, []int{1}[0]); }; _ = `+`; _ = '\\x2b'; _ = \";\" }")
	file := syntax.File{Src: source, Tokens: syntax.Scan(source)}
	for index := -1; index <= len(file.Tokens); index++ {
		for _, operator := range []byte("+-*/%&|^<>=!()[]{},;:.~") {
			got := tokCharIs(&file, index, operator)
			want := tokenTextIs(&file, index, string([]byte{operator}))
			if got != want {
				t.Fatalf("token=%d operator=%q: got %v, want %v", index, operator, got, want)
			}
		}
	}
}
