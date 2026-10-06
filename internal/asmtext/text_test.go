package asmtext

import (
	"strings"
	"testing"
)

func TestFunctionDiscoveryUsesDefinitionOrder(t *testing.T) {
	source := []byte(".global second,first\nfirst: ret\n/* ignore: fake */\nsecond: ret\n")
	lines, err := Scan(source)
	if err.Message != "" {
		t.Fatal(err)
	}
	functions, err := Functions(lines)
	if err.Message != "" || len(functions) != 2 || functions[0].Name != "first" || functions[1].Name != "second" || functions[0].Offset >= functions[1].Offset {
		t.Fatalf("discovery: %+v %+v", functions, err)
	}
	for _, bad := range []string{".globl missing", ".globl x\nx: ret\nx: ret", ".globl x,x\nx: ret"} {
		lines, err = Scan([]byte(bad))
		if err.Message != "" {
			t.Fatal(err)
		}
		if _, err = Functions(lines); err.Message == "" {
			t.Fatal("accepted", bad)
		}
	}
}

func TestLexicalWhitespaceAndUTF8Boundaries(t *testing.T) {
	for _, text := range []string{
		"", " \t\r\n\v\f", "  .global first, second  ",
		"\u0085\u00a0\u1680\u2000\u200a text \u2028\u2029\u202f\u205f\u3000",
		"\u00a0\xfftext\xff\u3000", "\xc2", "\u200btext\u200b",
	} {
		if got, want := trimSpace(text), strings.TrimSpace(text); got != want {
			t.Fatalf("trim %q: got %q, want %q", text, got, want)
		}
	}
	lines, err := Scan([]byte("\u00a0.global first, second\u3000\nfirst: ret; second: ret # ignored\n"))
	if err.Message != "" || len(lines) != 3 || lines[0].Text != ".global first, second" || lines[1].Text != "first: ret" || lines[2].Text != "second: ret" {
		t.Fatalf("line ownership/lexing: %+v %+v", lines, err)
	}
	functions, err := Functions(lines)
	if err.Message != "" || len(functions) != 2 || functions[0].Name != "first" || functions[1].Name != "second" {
		t.Fatalf("global discovery: %+v %+v", functions, err)
	}
	for _, bad := range []string{".global first,", ".global ,first", ".global first,,second"} {
		if _, err := Functions([]Line{{Text: bad}}); err.Message == "" {
			t.Fatal("accepted empty symbol", bad)
		}
	}
}
