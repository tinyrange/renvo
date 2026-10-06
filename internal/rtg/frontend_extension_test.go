//go:build !renvo

package rtg

import (
	"bytes"
	"strings"
	"testing"
)

type frontendExtensionLoader struct{ source []byte }

func (l frontendExtensionLoader) LoadImport(importer, name string) ImportSource {
	if name == "custom_ops.rtg" {
		return ImportSource{Source: l.source, Filename: "../../backends/custom_ops.rtg", Ok: true}
	}
	return (testFilesystemImportLoader{}).LoadImport(importer, name)
}

func TestFrontendOperationsExtensionPreservesPublicNames(t *testing.T) {
	base := append(frontendTestDefinition(t), []byte("\n@import \"custom_ops.rtg\"\n")...)
	extension := []byte(`extend arch i8086 {
	frontend_operations {
		custom_return() {
			lower = go msdos.emitReturn
			reads = [sp]
			writes = [sp]
			clobbers = []
			memory = read
			control = return
			ordering = ordered
			requires = []
		}
	}
}
`)
	r := Resolve(ParseImports(base, "../../backends/msdos.rtg", frontendExtensionLoader{extension}))
	if !r.Ok {
		t.Fatalf("extension resolve: %+v", r.Diagnostics)
	}
	v := FrontendOperations(r, "msdos/8086")
	if !v.Ok || len(v.Operations) != 10 {
		t.Fatalf("extension vocabulary: %+v", v)
	}
	a := ParseAssembly([]byte("rtgasm 2 assembly { answer(out:emitter) { custom_return() } }"), "answer.rtgasm")
	generated := GenerateAssemblyEvaluator(r, "msdos/8086", a, 0)
	if !generated.Ok || !bytes.Contains(generated.Source, []byte("EmitReturn")) {
		t.Fatalf("extension generation: %+v", generated.Diagnostics)
	}
	duplicate := strings.Replace(string(extension), "custom_return", "return", 1)
	bad := Resolve(ParseImports(base, "../../backends/msdos.rtg", frontendExtensionLoader{[]byte(duplicate)}))
	if bad.Ok || !hasDiagnosticCode(bad.Diagnostics, "RTG-FRONTEND-003") {
		t.Fatalf("extension override: %+v", bad.Diagnostics)
	}
}
