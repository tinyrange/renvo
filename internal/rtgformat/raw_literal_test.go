package rtgformat

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"testing"

	"renvo.dev/internal/rtg"
)

func TestSourcePreservesRawLiteralValuesAndIsIdempotent(t *testing.T) {
	input := []byte("go backend {\n// A comment containing ` must not start a literal.\nfunc text() string {\n_ = \"`\"\nreturn `first\n  spaces\n\ttab\n\nlast` + `\nsecond\n`\n}\n}\n")
	values := func(source []byte) []string {
		t.Helper()
		doc := rtg.Parse(source, "raw.rtg")
		var result []string
		for _, decl := range doc.Declarations {
			if decl.Kind != rtg.DeclGo {
				continue
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), "body.go", "package p\n"+string(source[decl.BodyStart:decl.BodyEnd]), 0)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(parsed, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					value, err := strconv.Unquote(lit.Value)
					if err != nil {
						t.Fatal(err)
					}
					result = append(result, value)
				}
				return true
			})
		}
		return result
	}
	want := values(input)
	if len(want) != 3 {
		t.Fatalf("fixture literals: %q", want)
	}
	formatted, err := Source(input, "raw.rtg", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := values(formatted); !reflect.DeepEqual(got, want) {
		t.Fatalf("literal values changed: got %q, want %q", got, want)
	}
	again, err := Source(formatted, "raw.rtg", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, formatted) {
		t.Fatalf("not idempotent:\n%s\nthen:\n%s", formatted, again)
	}
}
