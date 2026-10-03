package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestCoreDotImportsResolveForeignDeclarations(t *testing.T) {
	source := []byte(`package main
 import . "example.com/case/lib"
 type Alias = Thing
 var Initial = Counter
 func Read(v Thing) Thing { return v }
 func main() { print(Message()); _ = Message; var item Thing; _ = item; Counter = 23 }
 `)
	graph := genericTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/lib/lib.go", Src: []byte("package lib;func Message()int{return 17};type Thing struct{N int};var Counter=19")},
		{Path: "/repo/case/cmd/app/main.go", Src: source},
	})
	program := CheckGraphCore(graph)
	if !program.Ok {
		t.Fatalf("check: %#v", program)
	}
	root := len(program.Packages) - 1
	info := program.Packages[root]
	foreign := 0
	for _, body := range info.CoreBodies {
		for _, ref := range body.CoreRefs {
			if ref.Package != root {
				foreign++
				if ref.Package != 0 {
					t.Fatalf("foreign ref: %#v", ref)
				}
			}
		}
	}
	if foreign != 4 {
		t.Fatalf("foreign body references = %d, want 4", foreign)
	}
	for _, name := range []string{"Message()", "Thing", "Counter ="} {
		result := NavigateProgram(graph, program, "/repo/case/cmd/app/main.go", navigationTestOffset(source, name))
		if !result.Ok || result.Definition.Path != "/repo/case/lib/lib.go" {
			t.Fatalf("%s navigation: %#v", name, result)
		}
	}
}

func TestCoreDotImportsDoNotHideInvalidReferences(t *testing.T) {
	for _, tc := range []struct {
		name, dependency, source string
		want                     int
	}{
		{"undefined", "func Exported(){}", "func main(){ Missing() }", CheckErrUndefined},
		{"undefined_type", "type Thing int", "type Alias = Missing;func main(){}", CheckErrUndefined},
		{"private", "func private(){};func Exported(){}", "func main(){ private() }", CheckErrUndefined},
		{"arity", "func Exported(v int)int{return v}", "func main(){ Exported() }", CheckErrCallArity},
		{"operand", "func Exported()(int,int){return 1,2}", "func main(){ _ = Exported()+1 }", CheckErrOperand},
		{"private_field", "type Thing struct{private int}", "func main(){ _ = Thing{private:1} }", CheckErrUndefined},
		{"local_shadow", "func Exported()int{return 17}", "func main(){ Exported:=23;print(Exported) }", CheckOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{
				{Path: "/repo/case/lib/lib.go", Src: []byte("package lib;" + tc.dependency)},
				{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;import . \"example.com/case/lib\";" + tc.source)},
			})
			checked := CheckGraphCore(graph)
			if checked.Error != tc.want || checked.Ok != (tc.want == CheckOK) {
				t.Fatalf("check: ok=%v error=%d want=%d", checked.Ok, checked.Error, tc.want)
			}
		})
	}
}
