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
 var pointer *struct{ Value [N]int }
 var array [N]struct{ Value [N]int }
 var (
  grouped struct{ Value [N]int }
  nested []*struct{ Value [N]int }
 )
 { const N = 3; var c struct{ Value [N]int }; _ = c }
 _, _ = a, b
 _, _, _, _ = pointer, array, grouped, nested
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

func TestAnonymousInitializerAndSignatureLoweringRemainsEnabled(t *testing.T) {
	for _, source := range []string{
		`func main(){var(a = struct{Value int}{1});_=a}`,
		`func main(){var(a = &struct{Value int}{1});_=a}`,
		`func use(x struct{Value int}){};func main(){}`,
		`var a struct{Value int};func main(){}`,
	} {
		built := buildFromFiles(t, []load.SourceFile{
			{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
			{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)},
		})
		program := &built.Units[built.Root].Program
		if !lowerAnonymousTypes(program, false) || !bytes.Contains(program.Text, []byte("__renvo_anonymous_type_")) {
			t.Errorf("anonymous lowering lost for %s: %s", source, program.Text)
		}
	}
}
