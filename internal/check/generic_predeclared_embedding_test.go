package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestGenericPredeclaredEmbeddingPreservesGoChecking(t *testing.T) {
	for _, name := range []string{"bool", "string", "int", "uint", "uintptr", "byte", "rune", "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "complex64", "complex128", "any", "error"} {
		for _, pointer := range []bool{false, true} {
			if pointer && (name == "any" || name == "error") {
				continue
			}
			spelling := name
			if pointer {
				spelling = "*" + name
			}
			t.Run(spelling, func(t *testing.T) {
				source := []byte("package main;type Arg=struct{" + spelling + "};func Id[T any](v T)T{return v};func main(){var a Arg;b:=Id(a);_=b." + name + ";a=Arg{" + name + ":b." + name + "};_=a}")
				genericArrayLengthCheckGo(t, source)
				graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: source}})
				prepared := PrepareGenerics(graph)
				if !prepared.Ok {
					t.Fatalf("prepare: %s", prepared.Message)
				}
				checked := CheckGraphCore(prepared.Graph)
				if !checked.Ok {
					t.Fatalf("check: %+v\n%s", checked, prepared.Graph.Packages[0].Files[0].Src)
				}
				for _, pkg := range prepared.Graph.Packages {
					for _, file := range pkg.Files {
						genericArrayLengthCheckGo(t, file.Src)
					}
				}
			})
		}
	}
}

func TestGenericAnonymousPredeclaredEmbeddingPreservesGoChecking(t *testing.T) {
	for _, name := range []string{"bool", "int", "byte", "rune", "any", "error"} {
		t.Run(name, func(t *testing.T) {
			source := []byte("package main;func Id[T any](v T)T{return v};func main(){var a struct{" + name + "};b:=Id(a);keyed:=struct{" + name + "}{" + name + ":b." + name + "};_=Id(keyed)." + name + "}")
			genericArrayLengthCheckGo(t, source)
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: source}})
			prepared := PrepareGenerics(graph)
			if !prepared.Ok {
				t.Fatalf("prepare: %s", prepared.Message)
			}
			text := prepared.Graph.Packages[0].Files[0].Src
			genericArrayLengthCheckGo(t, text)
			if checked := CheckGraphCore(prepared.Graph); !checked.Ok {
				t.Fatalf("check: %+v\n%s", checked, text)
			}
		})
	}
}

func TestGenericMultilinePredeclaredEmbeddingPreservesGoChecking(t *testing.T) {
	source := []byte(`package main
 type Arg struct {
  Padding string
  int
 }
 func Id[T any](v T)T{return v}
 func main(){var a Arg;b:=Id(a);a=Arg{int:b.int};_=a
 var c struct {
  Padding string
  *int
 }
 d:=Id(c)
 _=d.int
 }
 `)
	genericArrayLengthCheckGo(t, source)
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: source}})
	prepared := PrepareGenerics(graph)
	if !prepared.Ok {
		t.Fatalf("prepare: %s", prepared.Message)
	}
	text := prepared.Graph.Packages[0].Files[0].Src
	genericArrayLengthCheckGo(t, text)
	if checked := CheckGraphCore(prepared.Graph); !checked.Ok {
		t.Fatalf("check: %+v\n%s", checked, text)
	}
}
