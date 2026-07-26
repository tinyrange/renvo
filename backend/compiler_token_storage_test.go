package main

import "testing"

func TestCompactTokensPreserveLargeRangesAndLines(t *testing.T) {
	var toks renvoTokens
	renvoScanAppendToken(&toks, renvoTokIdent, 7, 300, 1)
	renvoScanAppendToken(&toks, renvoTokIdent, 307, 2, 70000)
	renvoScanAppendToken(&toks, renvoTokEOF, 309, 0, 70001)

	p := &renvoProgram{src: make([]byte, 309), toks: toks}
	if got, want := len(toks.data), toks.count*2; got != want {
		t.Fatalf("compact token data length = %d, want %d", got, want)
	}
	if got := renvoTokStart(p, 0); got != 7 {
		t.Fatalf("large token start = %d, want 7", got)
	}
	if got := renvoTokEnd(p, 0); got != 307 {
		t.Fatalf("large token end = %d, want 307", got)
	}
	if got := renvoTokLine(p, 0); got != 1 {
		t.Fatalf("first token line = %d, want 1", got)
	}
	if got := renvoTokLine(p, 1); got != 70000 {
		t.Fatalf("large token line = %d, want 70000", got)
	}
	if got := renvoTokLine(p, 2); got != 70001 {
		t.Fatalf("following token line = %d, want 70001", got)
	}
}

func TestDecodedCompactTokensPreserveOperatorAndOverflowData(t *testing.T) {
	text := make([]byte, 303)
	text[300] = '+'
	var encoded []byte
	encoded = appendTestUnitVarint(encoded, 3)
	encoded = appendTestUnitToken(encoded, renvoTokIdent, 0, 300, 70000)
	encoded = appendTestUnitToken(encoded, renvoTokOp, 300, 1, 0)
	encoded = appendTestUnitToken(encoded, renvoTokEOF, 2, 0, 1)

	data, lineBases, ok := renvoDecodeUnitTokens(text, encoded)
	if !ok {
		t.Fatal("compact token decoder rejected valid overflow data")
	}
	p := &renvoProgram{
		src: text,
		toks: renvoTokens{
			data:      data,
			lineBases: lineBases,
			count:     3,
		},
	}
	if got := renvoTokEnd(p, 0); got != 300 {
		t.Fatalf("decoded large token end = %d, want 300", got)
	}
	if got := renvoTokLine(p, 0); got != 70000 {
		t.Fatalf("decoded large line = %d, want 70000", got)
	}
	if !renvoTokCharIs(p, 1, '+') {
		t.Fatal("decoded single-character operator lost its fast-path value")
	}
	if got := renvoTokLine(p, 2); got != 70001 {
		t.Fatalf("decoded following line = %d, want 70001", got)
	}
}

func appendTestUnitToken(dst []byte, kind int, startDelta int, size int, lineDelta int) []byte {
	dst = appendTestUnitVarint(dst, kind)
	dst = appendTestUnitVarint(dst, startDelta)
	dst = appendTestUnitVarint(dst, size)
	return appendTestUnitVarint(dst, lineDelta)
}

func appendTestUnitVarint(dst []byte, value int) []byte {
	for value >= 128 {
		dst = append(dst, byte(value&127|128))
		value >>= 7
	}
	return append(dst, byte(value))
}
