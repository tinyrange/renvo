//go:build renvo_bundle

package driver

import (
	"bytes"
	"testing"
	"testing/fstest"
)

func TestCompileCommandUsesBundledCLibrary(t *testing.T) {
	fs := memorySourceFS{files: fstest.MapFS{
		"main.c": {Data: []byte("#include <stdio.h>\nint main(void) { puts(\"hello\"); return 0; }\n")},
	}}
	result, err := CompileCommand(&CommandRequest{Filesystem: fs, Args: []string{"cc", "main.c"}, Target: "windows/386", ArenaSize: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ok || !bytes.HasPrefix(result.Binary, []byte("MZ")) {
		t.Fatalf("bundled C library compilation failed: %+v", result.Diagnostic)
	}
}
