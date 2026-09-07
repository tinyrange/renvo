package link

import (
	"bytes"
	"renvo.dev/internal/load"
	"testing"
)

func TestAnonymousLocalVariableTypesStayInScope(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
func main() {
 const N = 2
 var a, b struct{ Value [N]int }
 { const N = 3; var c struct{ Value [N]int }; _ = c }
 _, _ = a, b
}`)},
	})
	program := &built.Units[built.Root].Program
	if !lowerAnonymousTypes(program, false) {
		t.Fatal("anonymous lowering failed")
	}
	if bytes.Contains(program.Text, []byte("__renvo_anonymous_type_")) {
		t.Fatalf("local variable type escaped its scope: %s", program.Text)
	}
}
