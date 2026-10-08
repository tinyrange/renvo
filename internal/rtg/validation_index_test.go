package rtg

import (
	"bytes"
	"testing"
)

func TestResolveHookIndexDoesNotSurviveDeclarationEdits(t *testing.T) {
	source := frontendTestDefinition(t)
	document := ParseImports(source, "../../backends/msdos.rtg", testFilesystemImportLoader{})
	first := Resolve(document)
	if !first.Ok {
		t.Fatalf("initial resolve: %+v", first.Diagnostics)
	}
	// Resolve's document remains an editable public value. A local hook index
	// must not leak into it, nor into another resolution of the original.
	old := []byte("destination RTGRegister, value int")
	replacement := []byte("destination RTGRegister, value string")
	edited := false
	for i := range document.Declarations {
		decl := &document.Declarations[i]
		if decl.Kind == DeclGo && bytes.Contains(decl.GoSource, old) {
			decl.GoSource = bytes.Replace(decl.GoSource, old, replacement, 1)
			edited = true
			break
		}
	}
	if !edited {
		t.Fatal("missing hook signature in fixture")
	}
	for _, input := range []Document{document, first.Document} {
		bad := Resolve(input)
		if bad.Ok || !hasDiagnosticCode(bad.Diagnostics, "RTG-FRONTEND-004") {
			t.Fatalf("edited signature escaped validation: %+v", bad.Diagnostics)
		}
	}
}
