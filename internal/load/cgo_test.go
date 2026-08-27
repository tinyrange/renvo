package load

import (
	"bytes"
	"testing"

	"renvo.dev/internal/c11"
	"renvo.dev/internal/syntax"
)

func TestCgoExportHeader(t *testing.T) {
	file := syntax.ParseFile([]byte(`package bridge
/* int imported_value(void); */
import "C"

//export c_mix
func c_mix(left, right int, scale int32) int64 { return 0 }
`))
	if !file.Ok || len(file.Funcs) != 1 {
		t.Fatalf("parse = %#v", file)
	}
	definition := goExportDefinition{Mapping: c11.GoExport{CName: "c_mix", GoName: "c_mix"}, File: 0, Func: 0}
	header, ok := cgoExportHeader([]syntax.File{file}, []goExportDefinition{definition}, c11.DataModelLP64)
	if !ok || !bytes.Contains(header, []byte("int imported_value(void);")) ||
		!bytes.Contains(header, []byte("extern long long c_mix(long long, long long, int);")) {
		t.Fatalf("LP64 export header = %q, ok=%v", header, ok)
	}
	header, ok = cgoExportHeader([]syntax.File{file}, []goExportDefinition{definition}, c11.DataModelILP32)
	if !ok || !bytes.Contains(header, []byte("extern long long c_mix(int, int, int);")) {
		t.Fatalf("ILP32 export header = %q, ok=%v", header, ok)
	}
}

func TestCgoExportHeaderInclude(t *testing.T) {
	for _, src := range [][]byte{
		[]byte("#include \"_cgo_export.h\"\n"),
		[]byte("  # include <_cgo_export.h>\n"),
	} {
		if !cgoSourceIncludesExportHeader(src) {
			t.Fatalf("export header include not recognized: %q", src)
		}
	}
	if cgoSourceIncludesExportHeader([]byte("#include \"ordinary.h\"\n")) {
		t.Fatal("ordinary header recognized as the export header")
	}
}
