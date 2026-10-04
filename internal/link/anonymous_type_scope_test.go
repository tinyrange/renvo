package link

import (
	"bytes"
	wireunit "renvo.dev/backend/unit"
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

func TestAnonymousTypesRetainPrivateMemberPackageIdentity(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/left/lib.go", Src: []byte(`package left
func Value()any{return struct{value int}{42}}
func Accept(v interface{seal()int})int{return v.seal()}
`)},
		{Path: "/repo/case/right/lib.go", Src: []byte(`package right
func Value()any{return struct{value int}{42}}
func Accept(v interface{seal()int})int{return v.seal()}
`)},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
import "example.com/case/left"
import "example.com/case/right"
func main(){_,_=left.Value(),right.Value()}
`)},
	}
	for _, mode := range []string{"normal", "incremental", "transient", "incremental_transient"} {
		t.Run(mode, func(t *testing.T) {
			input := buildFromFiles(t, files)
			var result Result
			switch mode {
			case "incremental":
				result = LinkBuildCoreIncremental(input)
			case "transient":
				result = LinkBuildCoreTransient(input)
			case "incremental_transient":
				session := BeginPackageSession(input, true)
				for !session.Step() {
				}
				result = session.Result()
			default:
				result = LinkBuildCore(input)
			}
			if !result.Ok {
				t.Fatal("link failed")
			}
			program, err := wireunit.Unmarshal(result.Data)
			if err != nil {
				t.Fatal(err)
			}
			counts := map[string]int{}
			for _, declaration := range program.Decls {
				name := program.Text[declaration.NameStart:declaration.NameEnd]
				if !bytes.HasPrefix(name, []byte("__renvo_anonymous_type_")) {
					continue
				}
				for _, owner := range program.Packages {
					if declaration.NameStart >= owner.TextStart && declaration.NameStart < owner.TextEnd {
						counts[owner.ImportPath]++
					}
				}
			}
			if counts["example.com/case/left"] != 2 || counts["example.com/case/right"] != 2 || len(counts) != 2 {
				t.Fatalf("anonymous declaration owners: %v", counts)
			}
		})
	}
}
