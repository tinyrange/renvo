package driver

import (
	"go/constant"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestSourceEmbedQuotedExpressionPreservesAllBytes(t *testing.T) {
	for _, size := range []int{0, 256, 14999, 15000, 15001, 30000, 30001} {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i)
		}
		expression := string(quoteSourceEmbedExpression(data))
		value, err := types.Eval(token.NewFileSet(), nil, token.NoPos, expression)
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		if got := constant.StringVal(value.Value); got != string(data) {
			t.Fatalf("size %d: quoted expression changed input bytes", size)
		}
		chunks := 1
		if size > 15000 {
			chunks = (size + 14999) / 15000
		}
		if got := strings.Count(expression, " + "); got != chunks-1 {
			t.Fatalf("size %d: got %d separators, want %d", size, got, chunks-1)
		}
	}
}
