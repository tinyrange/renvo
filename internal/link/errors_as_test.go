package link

import (
	"bytes"
	"renvo.dev/internal/load"
	"renvo.dev/internal/unit"
	"testing"
)

func TestErrorsAsReferencesIncludeFunctionValues(t *testing.T) {
	for _, test := range []struct {
		body string
		want bool
	}{
		{"", false},
		{"_ = As(nil, nil)", true},
		{"callback := As; _ = callback", true},
	} {
		source := []byte("package main\nfunc asTarget(err error, target any) (bool,bool) { return false,false }\nfunc As(err error, target any) bool { if err == nil { return false }; return As(err,target) }\nfunc main() {" + test.body + "}\n")
		program := unit.Program{Package: "main"}
		if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
			t.Fatal("parse")
		}
		if got := errorsAsReferenced(&program, []string{"asTarget", "As"}); got != test.want {
			t.Fatalf("body %q: reference=%v, want %v", test.body, got, test.want)
		}
	}
}

func TestErrorsAsPackageObjectRetainsDispatch(t *testing.T) {
	program := unit.Program{Package: "errors", ImportPath: "errors"}
	if !errorsAsReferenced(&program, []string{"asTarget", "As"}) {
		t.Fatal("exported As needs dispatch")
	}
	if errorsAsReferenced(&program, nil) {
		t.Fatal("absent intrinsic cannot need dispatch")
	}
}

func TestErrorsAsDispatchSurvivesLinkModes(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/std/errors/errors.go", Src: []byte(`package errors
func asTarget(err error, target any) (bool,bool) { return false,false }
func As(err error, target any) bool { _,matched := asTarget(err,target); return matched }
`)},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
import "errors"
type detail struct{}
func (e *detail) Error() string { return "detail" }
func main() { var got *detail; if !errors.As(&detail{}, &got) { panic("As") } }
`)},
	}
	for _, mode := range []string{"normal", "incremental", "transient", "incremental_transient"} {
		t.Run(mode, func(t *testing.T) {
			input := buildFromFiles(t, files)
			var linked Result
			if mode == "incremental_transient" {
				session := BeginPackageSession(input, true)
				for !session.Step() {
				}
				linked = session.Result()
			} else if mode == "incremental" {
				linked = LinkBuildCoreIncremental(input)
			} else if mode == "transient" {
				linked = LinkBuildCoreTransient(input)
			} else {
				linked = LinkBuildCore(input)
			}
			if !linked.Ok {
				t.Fatalf("link: %d", linked.Error)
			}
			if !bytes.Contains(linked.Data, []byte("target.(**detail)")) {
				t.Fatal("missing concrete target dispatch")
			}
		})
	}
}
