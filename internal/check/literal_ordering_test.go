package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestLiteralComplexOrdering(t *testing.T) {
	for _, expression := range []string{
		"1i < 2i", "1i <= 2i", "1i > 2i", "1i >= 2i", "1 < 2i", "1i < 2",
		"(1i) < (2i)", "((1i)) < ((2i))", "1i + 2 < 3", "1 < 2 + 3i", "(1i - 1i) < 2", "-1i < +2i",
		"1i * 2 < 3", "1 < 2i / 3", "0i < 1", "'a' < 2i",
	} {
		t.Run(expression, func(t *testing.T) {
			for _, body := range []string{"_ = " + expression, "if " + expression + " {}", "_ = (" + expression + ")"} {
				graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc main() { " + body + " }\n")}})
				result := CheckGraphCore(graph)
				if result.Ok || result.Error != CheckErrOperand {
					t.Fatalf("%s: ok=%v error=%d, want operand diagnostic", body, result.Ok, result.Error)
				}
			}
		})
	}
}

func TestLiteralOrderingPreservesValidComparisons(t *testing.T) {
	for _, body := range []string{
		"_ = 1i == 2i", "_ = 1i != 2i", "_ = (1i - 1i) == 0", "_ = 1 < 2", "_ = 1.5 <= 2", "_ = 'a' > 'b'",
		"_ = 1 < 2 && 3 < 4", "_ = (1 < 2) == (3i == 4i)", "_ = real(1i) < 2", "_ = 1 < imag(2i)",
		"var x float64; _ = x < 0i", "var x float64; _ = x + 0i < 1", "var x float64; _ = 1 < 0i + x",
		"var x float64; _ = (x + 0i) < 1", "var x int; _ = x + (1i - 1i) < 2",
	} {
		t.Run(body, func(t *testing.T) {
			graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc main() { " + body + " }\n")}})
			result := CheckGraphCore(graph)
			if !result.Ok {
				t.Fatalf("error=%d token=%d", result.Error, result.ErrorToken)
			}
		})
	}
}
