package driver

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func TestScriptSourceKeepsImportsAndWrapsStatements(t *testing.T) {
	src := []byte(`// script comment
import (
	"fmt"
)

value := 42
fmt.Println(value)
`)
	wrapped := scriptSource(src)
	file := syntax.ParseFile(wrapped)
	if !file.Ok {
		t.Fatalf("wrapped script did not parse: err=%d token=%d\n%s", file.Error, file.ErrorTok, wrapped)
	}
	if len(file.Imports) != 1 || len(file.Funcs) != 1 {
		t.Fatalf("wrapped declarations = imports:%d funcs:%d\n%s", len(file.Imports), len(file.Funcs), wrapped)
	}
	name := syntax.TokenText(file.Src, file.Tokens[file.Funcs[0].NameTok])
	if string(name) != "main" {
		t.Fatalf("generated function = %q, want main", name)
	}
	if !bytes.Contains(wrapped, []byte("func main(){\n\nvalue := 42")) {
		t.Fatalf("script statements were not placed in main:\n%s", wrapped)
	}
}

func TestBuildUnitAcceptsOneScriptFile(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/hello.go", Src: []byte("value := 40 + 2\nif value == 42 { print(\"PASS\\n\") }\n")},
	}
	result := BuildUnit([]string{"-script", "-emit-unit", "-o", "hello.unit", "hello.go"}, "/repo/case", "/std", files)
	if !result.Ok {
		t.Fatalf("script build failed: %#v", result)
	}
	if len(result.Unit) < 4 || string(result.Unit[:4]) != "RNVO" {
		t.Fatalf("script unit has invalid header: %x", result.Unit)
	}
}

func TestParseOptionsScriptAndImage(t *testing.T) {
	options := ParseOptions([]string{"-script", "-emit-image", "-o", "hello.rnvi", "hello.go"})
	if !options.Ok || !options.Script || !options.EmitImage || len(options.Files) != 1 {
		t.Fatalf("script image options = %#v", options)
	}
	for _, tc := range []struct {
		args []string
		err  int
	}{
		{[]string{"-script", "-o", "out", "."}, ParseErrScriptRequiresFile},
		{[]string{"-script", "-o", "out", "a.go", "b.go"}, ParseErrScriptFileCount},
		{[]string{"-emit-unit", "-emit-image", "-o", "out", "a.go"}, ParseErrConflictingEmit},
	} {
		got := ParseOptions(tc.args)
		if got.Ok || got.Error != tc.err {
			t.Fatalf("ParseOptions(%q) = %#v, want error %d", tc.args, got, tc.err)
		}
	}
}

func TestStandaloneCSourceContextPreservesSingleGoObject(t *testing.T) {
	options := Options{
		Mode:    ModeObject,
		Package: "/repo/module/object.go",
		Files:   []string{"/repo/module/object.go"},
	}
	workDir, options := standaloneCSourceContext("/tmp/build", options)
	if workDir != "/repo/module" {
		t.Fatalf("object work directory = %q, want /repo/module", workDir)
	}
	if options.Package != "object.go" || len(options.Files) != 1 || options.Files[0] != "object.go" {
		t.Fatalf("object source options = %#v", options)
	}
}
