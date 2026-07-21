package syntax

import (
	"testing"
	"unsafe"
)

func TestTokenCompactLayout(t *testing.T) {
	tok := MakeToken(TokenIdent, 12, 18, 345)
	if tok.KindLine&255 != TokenIdent || TokenLine(tok) != 345 || tok.Start != 12 || tok.End != 18 {
		t.Fatalf("token fields were not preserved: %#v", tok)
	}
	if got, want := unsafe.Sizeof(tok), 3*unsafe.Sizeof(int(0)); got != want {
		t.Fatalf("token size = %d, want %d", got, want)
	}
	nonOperator := MakeToken(TokenIdent, 0, 1, int('+'))
	if tokCharIs(nil, []Token{nonOperator}, 0, '+') {
		t.Fatal("non-operator source line leaked into the packed operator field")
	}
}

func TestOperatorTokenPacking(t *testing.T) {
	src := []byte("\n\n{} +=")
	toks := Scan(src)
	if len(toks) != 4 {
		t.Fatalf("token count = %d, want 4", len(toks))
	}
	if TokenLine(toks[0]) != 3 || !tokCharIs(src, toks, 0, '{') || tokCharIs(src, toks, 0, '}') {
		t.Fatalf("opening operator token was not packed correctly: %#v", toks[0])
	}
	if TokenLine(toks[1]) != 3 || !tokCharIs(src, toks, 1, '}') {
		t.Fatalf("closing operator token was not packed correctly: %#v", toks[1])
	}
	if TokenLine(toks[2]) != 3 || tokCharIs(src, toks, 2, '+') {
		t.Fatalf("multi-byte operator token was not packed correctly: %#v", toks[2])
	}
	limit := MakeToken(TokenOperator, 0, 1, TokenLineLimit)
	limit.KindLine = limit.KindLine | int('{')<<TokenOperatorCharShift
	if TokenLine(limit) != TokenLineLimit {
		t.Fatalf("operator line limit = %d, want %d", TokenLine(limit), TokenLineLimit)
	}
}

func TestScanRejectsLinePastEncodingLimit(t *testing.T) {
	src := make([]byte, TokenLineLimit)
	for i := 0; i < len(src); i++ {
		src[i] = '\n'
	}
	var scanner Scanner
	scanner.Scan(src)
	if scanner.Ok {
		t.Fatal("scanner accepted a source line beyond the token encoding limit")
	}
}

func TestScanBasicFunction(t *testing.T) {
	src := []byte("package main\n\nfunc add(a int, b int) int { return a + b }\n")
	var scanner Scanner
	scanner.Scan(src)
	if !scanner.Ok {
		t.Fatal("scanner failed")
	}
	kinds := []int{
		TokenPackage, TokenIdent,
		TokenFunc, TokenIdent, TokenOperator, TokenIdent, TokenIdent, TokenOperator, TokenIdent, TokenIdent, TokenOperator, TokenIdent,
		TokenOperator, TokenReturn, TokenIdent, TokenOperator, TokenIdent, TokenOperator,
		TokenEOF,
	}
	if len(scanner.Tokens) != len(kinds) {
		t.Fatalf("token count = %d, want %d: %#v", len(scanner.Tokens), len(kinds), scanner.Tokens)
	}
	for i, kind := range kinds {
		if scanner.Tokens[i].KindLine&255 != kind {
			t.Fatalf("token %d kind = %d, want %d", i, scanner.Tokens[i].KindLine&255, kind)
		}
	}
	if string(TokenText(src, scanner.Tokens[3])) != "add" {
		t.Fatalf("function name token = %q", string(TokenText(src, scanner.Tokens[3])))
	}
}

func TestScanCommentsAndLiterals(t *testing.T) {
	src := []byte("package main\n// ignored\nvar s = \"a\\n\"\nvar r = `x\ny`\nvar c = '\\n'\n")
	var scanner Scanner
	scanner.Scan(src)
	if !scanner.Ok {
		t.Fatal("scanner failed")
	}
	foundString := 0
	foundChar := 0
	rawLine := 0
	for i := 0; i < len(scanner.Tokens); i++ {
		if scanner.Tokens[i].KindLine&255 == TokenString {
			foundString++
			if string(TokenText(src, scanner.Tokens[i])) == "`x\ny`" {
				rawLine = TokenLine(scanner.Tokens[i])
			}
		}
		if scanner.Tokens[i].KindLine&255 == TokenChar {
			foundChar++
		}
	}
	if foundString != 2 {
		t.Fatalf("string token count = %d, want 2", foundString)
	}
	if foundChar != 1 {
		t.Fatalf("char token count = %d, want 1", foundChar)
	}
	if rawLine != 4 {
		t.Fatalf("raw string line = %d, want 4", rawLine)
	}
}

func TestScanFrontendKeywords(t *testing.T) {
	src := []byte("break case chan const continue default defer else fallthrough for func go goto if import interface map package range return select struct switch type var")
	toks := Scan(src)
	want := []int{
		TokenBreak, TokenCase, TokenChan, TokenConst, TokenContinue,
		TokenDefault, TokenDefer, TokenElse, TokenFallthrough, TokenFor,
		TokenFunc, TokenGo, TokenGoto, TokenIf, TokenImport, TokenInterface,
		TokenMap, TokenPackage, TokenRange, TokenReturn, TokenSelect, TokenStruct,
		TokenSwitch, TokenType, TokenVar, TokenEOF,
	}
	if len(toks) != len(want) {
		t.Fatalf("token count = %d, want %d", len(toks), len(want))
	}
	for i := 0; i < len(want); i++ {
		if toks[i].KindLine&255 != want[i] {
			t.Fatalf("token %d kind = %d, want %d", i, toks[i].KindLine&255, want[i])
		}
	}
}

func TestScanKeywordLengthFastPathsRemainIdentifiers(t *testing.T) {
	src := []byte("i identifier tenletters fallthrougi")
	toks := Scan(src)
	if len(toks) != 5 {
		t.Fatalf("token count = %d, want 5", len(toks))
	}
	for i := 0; i < len(toks)-1; i++ {
		if toks[i].KindLine&255 != TokenIdent {
			t.Fatalf("token %d kind = %d, want identifier", i, toks[i].KindLine&255)
		}
	}
}

func TestScanNumericLiteralParity(t *testing.T) {
	src := []byte("077 0o77 0x1.8p+1 3e-2 1i 1.2e+3i")
	core := Scan(src)
	var scanner Scanner
	scanner.Scan(src)
	if !scanner.Ok || len(core) != len(scanner.Tokens) {
		t.Fatalf("numeric scan shape = core %d host %d ok %v", len(core), len(scanner.Tokens), scanner.Ok)
	}
	for i := 0; i < len(core); i++ {
		coreText := string(TokenText(src, core[i]))
		hostText := string(TokenText(src, scanner.Tokens[i]))
		if core[i].KindLine&255 != scanner.Tokens[i].KindLine&255 || coreText != hostText {
			t.Fatalf("numeric token %d = core %d %q host %d %q", i, core[i].KindLine&255, coreText, scanner.Tokens[i].KindLine&255, hostText)
		}
	}
}

func TestKeywordHashCollisionRemainsIdentifier(t *testing.T) {
	src := []byte("bits chan")
	toks := Scan(src)
	if len(toks) != 3 {
		t.Fatalf("token count = %d, want 3", len(toks))
	}
	if toks[0].KindLine&255 != TokenIdent {
		t.Fatalf("bits token kind = %d, want identifier", toks[0].KindLine&255)
	}
	if toks[1].KindLine&255 != TokenChan {
		t.Fatalf("chan token kind = %d, want chan", toks[1].KindLine&255)
	}
}

func TestScanInvalidString(t *testing.T) {
	var scanner Scanner
	scanner.Scan([]byte("package main\nvar s = \"unterminated\n"))
	if scanner.Ok {
		t.Fatal("scanner succeeded on invalid string")
	}
}

func TestStringLiteralValue(t *testing.T) {
	src := []byte("package main\nvar a = \"x\\ny\\x2fz\"\nvar b = `raw/path`\nvar c = \"\\u0214\\U0001F642\\141\\a\\b\\f\\v\"\n")
	toks := Scan(src)
	found := 0
	for i := 0; i < len(toks); i++ {
		if toks[i].KindLine&255 != TokenString {
			continue
		}
		value, ok := StringLiteralValue(src, toks[i])
		if !ok {
			t.Fatalf("StringLiteralValue failed for %q", string(TokenText(src, toks[i])))
		}
		if found == 0 && value != "x\ny/z" {
			t.Fatalf("first value = %q, want x\\ny/z", value)
		}
		if found == 1 && value != "raw/path" {
			t.Fatalf("second value = %q, want raw/path", value)
		}
		if found == 2 && value != "Ȕ🙂a\a\b\f\v" {
			t.Fatalf("third value = %q, want decoded Unicode, octal, and simple escapes", value)
		}
		found++
	}
	if found != 3 {
		t.Fatalf("string count = %d, want 3", found)
	}
}

func TestScanRejectsInvalidStringEscapes(t *testing.T) {
	invalid := []string{
		"package main\nvar s = \"\\q\"\n",
		"package main\nvar s = \"\\400\"\n",
		"package main\nvar s = \"\\uD800\"\n",
		"package main\nvar s = \"\\U00110000\"\n",
	}
	for _, src := range invalid {
		var scanner Scanner
		scanner.Scan([]byte(src))
		if scanner.Ok {
			t.Fatalf("scanner accepted invalid string escape in %q", src)
		}
	}
}
