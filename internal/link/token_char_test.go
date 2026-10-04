package link

import (
	"testing"

	"renvo.dev/internal/syntax"
	"renvo.dev/internal/unit"
)

func TestTokenCharMatchesStringComparison(t *testing.T) {
	program := unit.Program{Text: []byte{'(', 0, 255, ')'}}
	for _, span := range [][2]int{{0, 1}, {1, 1}, {2, 1}, {3, 1}, {-1, 1}, {4, 1}, {0, 0}, {0, 2}, {3, 2}} {
		program.Tokens = []unit.Token{{Start: span[0], Size: span[1]}}
		for _, index := range []int{-1, 0, 1} {
			for _, value := range []byte{'(', ')', 0, 255} {
				got := functionValueTokenCharIs(&program, index, value)
				want := functionValueTokenEquals(&program, index, string([]byte{value}))
				if got != want {
					t.Fatalf("span=%v index=%d value=%d: got %v, want %v", span, index, value, got, want)
				}
			}
		}
	}
}

func TestTokenTextEqualsPreservesEmptyAndInvalidSpans(t *testing.T) {
	program := unit.Program{Text: []byte("a\x00\xffb")}
	for _, span := range [][2]int{{0, 1}, {1, 1}, {2, 1}, {0, 4}, {4, 0}, {-1, 1}, {3, 2}} {
		program.Tokens = []unit.Token{{Start: span[0], Size: span[1]}}
		for _, index := range []int{-1, 0, 1} {
			for _, want := range []string{"", "a", "b", "\x00", "\xff", "a\x00\xffb"} {
				if got := functionValueTokenTextEquals(&program, index, want); got != (functionValueTokenText(&program, index) == want) {
					t.Fatalf("span=%v index=%d want=%q: got %v", span, index, want, got)
				}
			}
		}
	}
}

func TestTokenAtSpanSearchBoundaries(t *testing.T) {
	maximum := int(^uint(0) / 2)
	program := unit.Program{Tokens: []unit.Token{{Start: 0, Size: 1}, {Start: 1}, {Start: 1, Size: 2}, {Start: 4, Size: 1}, {Start: maximum - 2, Size: 1}}}
	for _, test := range [][3]int{{-1, 0, -1}, {0, 1, 0}, {1, 1, 1}, {1, 3, 2}, {2, 3, -1}, {4, 5, 3}, {maximum - 2, maximum - 1, 4}, {maximum, maximum, -1}} {
		if got := functionValueTokenAtSpan(&program, test[0], test[1]); got != test[2] {
			t.Fatalf("span=%v: got %d, want %d", test[:2], got, test[2])
		}
	}
}

func TestTokenKindMatchesParsedKeywords(t *testing.T) {
	source := []byte(`package const var type func struct return if else for break continue goto switch case default function "func"`)
	program := unit.Program{Text: source}
	for _, token := range syntax.Scan(source) {
		program.Tokens = append(program.Tokens, unit.MakeToken(functionValueUnitTokenKind(source, token), syntax.TokenStart(token), syntax.TokenSize(token), syntax.TokenLine(token)))
	}
	keywords := map[string]int{"package": unit.TokenPackage, "const": unit.TokenConst, "var": unit.TokenVar, "type": unit.TokenType, "func": unit.TokenFunc, "struct": unit.TokenStruct, "return": unit.TokenReturn, "if": unit.TokenIf, "else": unit.TokenElse, "for": unit.TokenFor, "break": unit.TokenBreak, "continue": unit.TokenContinue, "goto": unit.TokenGoto, "switch": unit.TokenSwitch, "case": unit.TokenCase, "default": unit.TokenDefault}
	for index := -1; index <= len(program.Tokens); index++ {
		for word, kind := range keywords {
			if got, want := functionValueTokenKindIs(&program, index, kind), functionValueTokenEquals(&program, index, word); got != want {
				t.Fatalf("token=%d keyword=%q: got %v, want %v", index, word, got, want)
			}
		}
	}
}
