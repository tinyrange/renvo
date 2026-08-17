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

func TestInvalidConcurrencyStatements(t *testing.T) {
	cases := []struct {
		name string
		body string
		err  int
	}{
		{"go requires call", "go len(\"value\")", CheckErrGoroutine},
		{"select case communicates", "select { case value: println(value) }", CheckErrSelect},
		{"one default", "select { default: println(1); default: println(2) }", CheckErrSelect},
		{"valid", "ch := make(chan int, 1); go close(ch); select { case value := <-ch: _ = value; default: }", CheckOK},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			file := syntax.ParseFile([]byte("package main\nfunc main() { " + test.body + " }\n"))
			if !file.Ok || len(file.Funcs) != 1 {
				t.Fatalf("parse failed: %#v", file)
			}
			body := syntax.ParseFuncBody(file, file.Funcs[0])
			got, _ := invalidDefiniteStatement(file, body)
			if got != test.err {
				t.Fatalf("error = %d, want %d", got, test.err)
			}
		})
	}
}

func TestChannelValidationDoesNotTreatSelectorCloseAsBuiltin(t *testing.T) {
	file := syntax.ParseFile([]byte("package main\ntype input struct{}\nfunc (input) close() {}\nfunc main() { var value input; value.close() }\n"))
	if !file.Ok || len(file.Funcs) != 2 {
		t.Fatalf("parse failed: %#v", file)
	}
	if token := invalidDefiniteChannelOperation(file, file.Funcs[1]); token >= 0 {
		t.Fatalf("selector close rejected at token %d", token)
	}
}

func TestChannelDirectionValidationUsesLexicalBinding(t *testing.T) {
	source := []byte(`package main
type Send chan<- int
type Receive <-chan int
var global Send
func send(values Send) { values <- 1 }
func receive(values Receive) { _ = <-values }
func narrow(values chan int) { var sendOnly chan<- int; var receiveOnly <-chan int; sendOnly = values; receiveOnly = values; _, _ = sendOnly, receiveOnly }
func invalid() { _ = <-global }
`)
	file := syntax.ParseFile(source)
	if !file.Ok || len(file.Funcs) != 4 {
		t.Fatalf("parse failed: %#v", file)
	}
	if tok := invalidDefiniteChannelOperation(file, file.Funcs[0]); tok >= 0 {
		t.Fatalf("valid send rejected at token %d", tok)
	}
	if tok := invalidDefiniteChannelOperation(file, file.Funcs[1]); tok >= 0 {
		t.Fatalf("valid receive rejected at token %d", tok)
	}
	if tok := invalidDefiniteChannelOperation(file, file.Funcs[2]); tok >= 0 {
		t.Fatalf("valid direction narrowing rejected at token %d", tok)
	}
	if tok := invalidDefiniteChannelOperation(file, file.Funcs[3]); tok < 0 {
		t.Fatal("receive from named send-only global was accepted")
	}
}

func TestChannelValidationPreservesIntegerCloseSyscall(t *testing.T) {
	file := syntax.ParseFile([]byte("package main\nfunc main() { descriptor := 3; close(descriptor) }\n"))
	if !file.Ok || len(file.Funcs) != 1 {
		t.Fatal("parse failed")
	}
	if tok := invalidDefiniteChannelOperation(file, file.Funcs[0]); tok >= 0 {
		t.Fatalf("integer close syscall rejected at token %d", tok)
	}
}
