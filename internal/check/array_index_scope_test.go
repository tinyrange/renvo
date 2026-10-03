package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestArrayIndexPackageDeclarationScope(t *testing.T) {
	for _, index := range []string{"2", "3"} {
		for _, separate := range []bool{false, true} {
			// The local length shadows the package constant, but must not affect the
			// package variable's array length (including when its type is in another file).
			declarations := "const length = 2\nvar data [length+1]byte\n"
			function := "func check() { const length = 99; _ = data[" + index + "] }\n"
			files := []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + function + declarations)}}
			if separate {
				files[0].Src = []byte("package main\n" + function)
				files = append(files, load.SourceFile{Path: "/repo/case/cmd/app/data.go", Src: []byte("package main\n" + declarations)})
			}
			program := CheckGraphCore(checkTestGraph(t, files))
			if index == "3" {
				if program.Ok || program.Error != CheckErrArrayIndex {
					t.Fatalf("separate=%v: expected array index error: %#v", separate, program)
				}
			} else if !program.Ok {
				t.Fatalf("separate=%v: rejected valid index: %#v", separate, program)
			}
		}
	}
}
