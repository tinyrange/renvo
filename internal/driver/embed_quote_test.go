package driver

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"
)

func TestEmbedQuotedExpressionRoundTrip(t *testing.T) {
	for _, size := range []int{0, 1, 255, 15000, 15001, 45017} {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i)
		}
		expression := quoteSourceEmbedExpression(data)
		parsed, err := parser.ParseExpr(string(expression))
		if err != nil {
			t.Fatal(err)
		}
		var decoded []byte
		var visit func(ast.Expr)
		visit = func(expr ast.Expr) {
			switch e := expr.(type) {
			case *ast.ParenExpr:
				visit(e.X)
			case *ast.BinaryExpr:
				if e.Op != token.ADD {
					t.Fatalf("unexpected operator %s", e.Op)
				}
				visit(e.X)
				visit(e.Y)
			case *ast.BasicLit:
				value, err := strconv.Unquote(e.Value)
				if err != nil {
					t.Fatal(err)
				}
				decoded = append(decoded, value...)
			default:
				t.Fatalf("unexpected expression %T", expr)
			}
		}
		visit(parsed)
		if !bytes.Equal(decoded, data) {
			t.Fatalf("size %d: quoted expression changed the bytes", size)
		}
		if cap(expression) != len(expression) {
			t.Fatalf("size %d: quoted buffer grew or over-reserved: %d/%d", size, len(expression), cap(expression))
		}
	}
}
