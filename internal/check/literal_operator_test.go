package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestLiteralUnaryOperands(t *testing.T) {
	for _, expression := range []string{
		"!1", "!(1)", "!((1))", "!1.2", "!1i", "!'a'", "!\"x\"",
		"*1", "*(1)", "&1", "&'a'", "+\"x\"", "-\"x\"", "^\"x\"", "^1.2", "^(1.2)",
	} {
		t.Run(expression, func(t *testing.T) {
			graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc main() { _ = " + expression + " }\n")}})
			result := CheckGraphCore(graph)
			if result.Ok || result.Error != CheckErrOperand {
				t.Fatalf("ok=%v error=%d, want operand diagnostic", result.Ok, result.Error)
			}
		})
	}
}

func TestLiteralUnaryPreservesValidExpressions(t *testing.T) {
	for _, expression := range []string{
		"1 * 2", "1 & 2", "1 ^ 2", "1 + 2", "1 - 2", "^1", "^'a'", "-1", "+1.2", "-1i",
		"!(1 == 2)", "!((1 == 2))", "!((1) == 2)", "!(((1)) == 2)", "!((\"x\")[0] == 1)", "+\"x\"[0]", "-(\"x\")[0]", "-((\"x\"))[0]",
	} {
		t.Run(expression, func(t *testing.T) {
			graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc main() { _ = " + expression + " }\n")}})
			result := CheckGraphCore(graph)
			if !result.Ok {
				t.Fatalf("error=%d token=%d", result.Error, result.ErrorToken)
			}
		})
	}
}
