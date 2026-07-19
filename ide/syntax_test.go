package ide

import "testing"

func TestHighlightGoLineClassifiesOrdinarySource(t *testing.T) {
	line := "func main() { value := 42; println(\"hello\") // note"
	spans, state := highlightGoLine(line, goSyntaxNormal)
	if state != goSyntaxNormal {
		t.Fatalf("state = %d", state)
	}
	assertSyntaxKindAt(t, line, spans, "func", syntaxKeyword)
	assertSyntaxKindAt(t, line, spans, "value", syntaxText)
	assertSyntaxKindAt(t, line, spans, "42", syntaxNumber)
	assertSyntaxKindAt(t, line, spans, "println", syntaxBuiltin)
	assertSyntaxKindAt(t, line, spans, "\"hello\"", syntaxString)
	assertSyntaxKindAt(t, line, spans, "// note", syntaxComment)
}

func TestHighlightGoLineCarriesMultilineState(t *testing.T) {
	first, state := highlightGoLine("var s = `raw", goSyntaxNormal)
	assertSyntaxKindAt(t, "var s = `raw", first, "`raw", syntaxString)
	if state != goSyntaxRawString {
		t.Fatalf("raw state = %d", state)
	}
	secondLine := "text` /* block"
	second, state := highlightGoLine(secondLine, state)
	assertSyntaxKindAt(t, secondLine, second, "text`", syntaxString)
	assertSyntaxKindAt(t, secondLine, second, "/* block", syntaxComment)
	if state != goSyntaxBlockComment {
		t.Fatalf("block state = %d", state)
	}
	thirdLine := "continued */ return"
	third, state := highlightGoLine(thirdLine, state)
	assertSyntaxKindAt(t, thirdLine, third, "continued */", syntaxComment)
	assertSyntaxKindAt(t, thirdLine, third, "return", syntaxKeyword)
	if state != goSyntaxNormal {
		t.Fatalf("final state = %d", state)
	}
}

func assertSyntaxKindAt(t *testing.T, line string, spans []syntaxSpan, text string, want syntaxKind) {
	t.Helper()
	start := -1
	for i := 0; i+len(text) <= len(line); i++ {
		if line[i:i+len(text)] == text {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("%q not found in %q", text, line)
	}
	for i := 0; i < len(spans); i++ {
		if start >= spans[i].start && start < spans[i].end {
			if spans[i].kind != want {
				t.Fatalf("%q kind = %d, want %d; spans = %#v", text, spans[i].kind, want, spans)
			}
			return
		}
	}
	t.Fatalf("%q was not covered; spans = %#v", text, spans)
}
