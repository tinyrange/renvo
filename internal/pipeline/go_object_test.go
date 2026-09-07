package pipeline

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestGoObjectExportsPackageRoots(t *testing.T) {
	for _, tc := range []struct {
		source       string
		want, absent []string
	}{
		{"package p\nfunc Add(a,b int)int{return helper(a)+b}\nfunc helper(a int)int{return a}\n", []string{"//export Add\n"}, []string{"//export helper"}},
		{"package p\n//export public_add\nfunc Add(a,b int)int{return a+b}\n", []string{"//export public_add\n"}, []string{"//export Add"}},
		{"package main\nfunc main(){}\n", []string{"//export main\nfunc appMain"}, nil},
		{"package main\n//export main\nfunc main(){}\n", []string{"//export main\nfunc main"}, []string{"//export main\nfunc appMain"}},
		{"package p\ntype T int\nfunc(t T) Method()int{return 1}\n", nil, []string{"//export Method"}},
		{"package p\nfunc Écho()int{return 1}\n", []string{"//export Écho\n"}, nil},
	} {
		result := BuildObjectUnit("/repo/case", "/std", ".", []load.SourceFile{{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.22\n")}, {Path: "/repo/case/main.go", Src: []byte(tc.source)}})
		if !result.Ok {
			t.Fatalf("%s: pipeline=%d build=%#v", tc.source, result.Error, result.Build)
		}
		for _, text := range tc.want {
			if !bytes.Contains(result.Link.Program.Text, []byte(text)) {
				t.Errorf("missing %q in %s", text, result.Link.Program.Text)
			}
		}
		for _, text := range tc.absent {
			if bytes.Contains(result.Link.Program.Text, []byte(text)) {
				t.Errorf("unexpected %q in %s", text, result.Link.Program.Text)
			}
		}
	}
}
