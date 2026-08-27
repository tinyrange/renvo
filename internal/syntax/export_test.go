package syntax

import "testing"

func TestExportDirective(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "adjacent", src: "package p\n//export c_name\nfunc goName() {}\n", want: "c_name"},
		{name: "indented", src: "package p\n  //export c_name\n  func goName() {}\n", want: "c_name"},
		{name: "blank line", src: "package p\n//export c_name\n\nfunc goName() {}\n"},
		{name: "trailing text", src: "package p\n//export c_name extra\nfunc goName() {}\n"},
		{name: "invalid name", src: "package p\n//export 2name\nfunc goName() {}\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file := ParseFile([]byte(test.src))
			if !file.Ok || len(file.Funcs) != 1 {
				t.Fatalf("parse = %#v", file)
			}
			if got := ExportDirective(file, file.Funcs[0]); got != test.want {
				t.Fatalf("ExportDirective() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestExportDirectiveRejectsMethod(t *testing.T) {
	file := ParseFile([]byte("package p\ntype T struct{}\n//export method\nfunc (T) Method() {}\n"))
	if !file.Ok || len(file.Funcs) != 1 {
		t.Fatalf("parse = %#v", file)
	}
	if got := ExportDirective(file, file.Funcs[0]); got != "" {
		t.Fatalf("method export = %q", got)
	}
}
