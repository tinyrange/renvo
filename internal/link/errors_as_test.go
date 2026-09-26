package link

import (
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
