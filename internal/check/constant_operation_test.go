package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestInvalidPackageConstantOperations(t *testing.T) {
	for _, source := range []string{
		"const x = 1/0",
		"const x = 1%0",
		"const n = -1; const x = 1<<n",
		"const n = -1; const x = 1>>n",
		"const x = 1<<(-2+1)",
		"const x = 1/(3-3)",
		"const x = 1/(2*(1-1))",
		"const x = -(1/0)",
		"const x = int(1/0)",
		"const x = 1/n; const n = z; const z = 0",
		"const x, y = 1, 1/0",
	} {
		t.Run(source, func(t *testing.T) {
			graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source + "\nfunc main(){}")}})
			result := CheckGraphCore(graph)
			if result.Ok || result.Error != CheckErrConstantOperation {
				t.Fatalf("ok=%v error=%d, want invalid constant operation", result.Ok, result.Error)
			}
		})
	}
}

func TestValidPackageConstantOperations(t *testing.T) {
	for _, source := range []string{
		"const x = 1/(3-2)",
		"const n = 2; const x = 1<<n",
		"const n = ^uint8(0); const x = 1<<n",
		"const n uint8 = 2; const x = 1<<n",
		"const n = 1<<100", // unknown, not a negative wrapped value
		"const x = 1/(1<<64)",
		"const x = 1/(9223372036854775807+1)",
		"const x = 1/(4611686018427387904*4)",
		"const n = -1; func f(n int) int { return 1<<n }",
		"const n = 0; func f(n int) int { return 1/n }",
		"func f() { const n = -1 }; const n = 2; const m = n; const x = 1<<m",
	} {
		t.Run(source, func(t *testing.T) {
			graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source + "\nfunc main(){}")}})
			if result := CheckGraphCore(graph); !result.Ok {
				t.Fatalf("error=%d token=%d", result.Error, result.ErrorToken)
			}
		})
	}
}

func TestConstantIndexArithmeticDoesNotWrap(t *testing.T) {
	maximum := int(^uint(0) >> 1)
	minimum := -maximum - 1
	for _, tc := range []struct {
		op          string
		left, right int
	}{
		{"+", maximum, 1}, {"+", minimum, -1},
		{"-", minimum, 1}, {"-", maximum, -1},
		{"*", maximum, 2}, {"*", minimum, -1},
		{"/", minimum, -1}, {"<<", maximum, 1},
		{"<<", 1, -1}, {">>", 1, -1},
	} {
		if value, ok := applyConstantIndexOperator(tc.op, tc.left, tc.right); ok {
			t.Fatalf("%d %s %d wrapped to %d", tc.left, tc.op, tc.right, value)
		}
	}
}
