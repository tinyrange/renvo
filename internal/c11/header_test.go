package c11

import (
	"bytes"
	"testing"
)

type headerTestReader struct {
	name string
	src  []byte
}

func (r headerTestReader) ReadInclude(_ string, name string, angled bool) ([]byte, string, bool) {
	if !angled || name != r.name {
		return nil, "", false
	}
	return r.src, "/usr/include/" + name, true
}

func TestBuildObjectPreludeRetainsOnlyReferencedHeaderDeclaration(t *testing.T) {
	source := []byte("#include <stdio.h>\nint main(void) { puts(\"PASS\"); return 0; }\n")
	header := []byte("extern int remove (const char *__name);\nextern int puts (const char *__s);\n")
	result := BuildObjectPrelude("/tmp/main.c", source, headerTestReader{name: "stdio.h", src: header})
	if !result.Ok || !bytes.Equal(result.Prelude, []byte("extern int puts ( const char * __s );\n")) {
		t.Fatalf("BuildObjectPrelude = %#v, prelude %q", result, result.Prelude)
	}
}

func TestBuildObjectPreludeReportsMissingHeaderAtDirective(t *testing.T) {
	result := BuildObjectPrelude("/tmp/main.c", []byte("  #include <missing.h>\nint main(void) { return 0; }\n"), headerTestReader{})
	if result.Ok || result.ErrorPath != "missing.h" || result.ErrorAt != 2 {
		t.Fatalf("BuildObjectPrelude missing header = %#v", result)
	}
}
