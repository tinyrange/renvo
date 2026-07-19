package check

import (
	"testing"

	"renvo.dev/internal/syntax"
)

func TestInvalidDefiniteStatements(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{name: "non-function call", body: "x := 1; x()", want: CheckErrCall},
		{name: "assignment target", body: "1 = 2", want: CheckErrAssignTarget},
		{name: "assignment count", body: "a, b := 1; _, _ = a, b", want: CheckErrAssignCount},
		{name: "bare break", body: "break", want: CheckErrBreak},
		{name: "bare continue", body: "continue", want: CheckErrContinue},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := syntax.ParseFile([]byte("package main\nfunc main() { " + tc.body + " }\n"))
			if !file.Ok || len(file.Funcs) != 1 {
				t.Fatalf("parse failed: %#v", file)
			}
			body := syntax.ParseFuncBody(file, file.Funcs[0])
			got, tok := invalidDefiniteStatement(file, body)
			if got != tc.want || tok < 0 {
				t.Fatalf("validation = (%d, %d), want error %d", got, tok, tc.want)
			}
		})
	}
}

func TestDefiniteStatementValidationPreservesValidForms(t *testing.T) {
	source := []byte(`package main

type item struct { value int }

func pair() (int, int) { return 1, 2 }

func main() {
	f := func() {}
	f()
	a, b := pair()
	m := map[string]int{"a": 1}
	v, ok := m["a"]
	p := &a
	*p = b
	(*p)++
	values := []int{a}
	values[0] = v
	x := item{}
	x.value = values[0]
	for {
		continue
		break
	}
	switch ok {
	case true:
		break
	}
}
`)
	file := syntax.ParseFile(source)
	if !file.Ok {
		t.Fatalf("parse failed: %#v", file)
	}
	body := syntax.ParseFuncBody(file, file.Funcs[1])
	if code, tok := invalidDefiniteStatement(file, body); code != CheckOK {
		t.Fatalf("valid statements rejected: error=%d token=%d text=%q", code, tok, syntax.TokenText(file.Src, file.Tokens[tok]))
	}
}
