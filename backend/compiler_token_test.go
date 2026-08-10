package main

import "testing"

func TestTokenPunctuationRequiresOperatorKind(t *testing.T) {
	p := renvoProgram{toks: renvoTokens{
		data: []int32{
			int32(renvoTokString | int('[')<<24), 0,
			int32(renvoTokOp | int('[')<<24), 0,
		},
		count: 2,
	}}
	if got := renvoTokSingleChar(&p, 0); got != 0 {
		t.Fatalf("large string token reported punctuation %q", got)
	}
	if renvoTokCharIs(&p, 0, '[') {
		t.Fatal("large string token matched '[' punctuation")
	}
	if got := renvoTokSingleChar(&p, 1); got != '[' || !renvoTokCharIs(&p, 1, '[') {
		t.Fatalf("operator punctuation = %q, matched %v", got, renvoTokCharIs(&p, 1, '['))
	}
}
