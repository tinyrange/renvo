package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestCFunctionsDoNotUseGoTerminatingStatementRules(t *testing.T) {
	for _, source := range []string{
		"int main(void) { for (;;) {} }",
		"int forever(void) { while (1) {} } int main(void) { return 0; }",
		"int optional(int condition) { if (condition) return 1; } int main(void) { return 0; }",
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.c", Src: []byte(source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Errorf("C source rejected: %s: error=%d token=%d", source, result.Error, result.ErrorToken)
		}
	}
}
