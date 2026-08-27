package syntax

import "testing"

func TestCgoPreamble(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
		ok   bool
	}{
		{name: "block", src: "package p\n/*\nint value(void);\n*/\nimport \"C\"\n", want: "\nint value(void);\n", ok: true},
		{name: "line", src: "package p\n// int first(void);\nimport \"C\"\n", want: " int first(void);", ok: true},
		{name: "no comment", src: "package p\nimport \"C\"\n", want: "", ok: true},
		{name: "blank line", src: "package p\n// int value(void);\n\nimport \"C\"\n", want: "", ok: true},
		{name: "ordinary import", src: "package p\nimport \"fmt\"\n", ok: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file := ParseFile([]byte(test.src))
			if !file.Ok || len(file.Imports) != 1 {
				t.Fatalf("parse = %#v", file)
			}
			got, _, ok := CgoPreamble(file, file.Imports[0])
			if ok != test.ok || string(got) != test.want {
				t.Fatalf("CgoPreamble() = %q, _, %v; want %q, _, %v", got, ok, test.want, test.ok)
			}
		})
	}
}

func TestCgoPreambleRejectsGroupedImport(t *testing.T) {
	file := ParseFile([]byte("package p\n/* int value(void); */\nimport (\n\t\"C\"\n)\n"))
	if !file.Ok || len(file.Imports) != 1 {
		t.Fatalf("parse = %#v", file)
	}
	if _, _, ok := CgoPreamble(file, file.Imports[0]); ok {
		t.Fatal("grouped import unexpectedly has a cgo preamble")
	}
}
