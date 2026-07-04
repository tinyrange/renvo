package syntax

import "testing"

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
		if scanner.Tokens[i].Kind != kind {
			t.Fatalf("token %d kind = %d, want %d", i, scanner.Tokens[i].Kind, kind)
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
		if scanner.Tokens[i].Kind == TokenString {
			foundString++
			if string(TokenText(src, scanner.Tokens[i])) == "`x\ny`" {
				rawLine = scanner.Tokens[i].Line
			}
		}
		if scanner.Tokens[i].Kind == TokenChar {
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
	src := []byte("interface map defer go select chan fallthrough")
	toks := Scan(src)
	want := []int{TokenInterface, TokenMap, TokenDefer, TokenGo, TokenSelect, TokenChan, TokenFallthrough, TokenEOF}
	if len(toks) != len(want) {
		t.Fatalf("token count = %d, want %d", len(toks), len(want))
	}
	for i := 0; i < len(want); i++ {
		if toks[i].Kind != want[i] {
			t.Fatalf("token %d kind = %d, want %d", i, toks[i].Kind, want[i])
		}
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
	src := []byte("package main\nvar a = \"x\\ny\\x2fz\"\nvar b = `raw/path`\n")
	toks := Scan(src)
	found := 0
	for i := 0; i < len(toks); i++ {
		if toks[i].Kind != TokenString {
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
		found++
	}
	if found != 2 {
		t.Fatalf("string count = %d, want 2", found)
	}
}
