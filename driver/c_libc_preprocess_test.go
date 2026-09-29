//go:build renvo_bundle

package driver

import (
	"bytes"
	"testing"
	"testing/fstest"
)

func TestPreprocessBundledLibc(t *testing.T) {
	fs := memorySourceFS{files: fstest.MapFS{"probe.c": {Data: []byte("#include <wchar.h>\n#include <locale.h>\n#include <errno.h>\nmbstate_t state;\n")}}}
	for _, noStandard := range []bool{false, true} {
		args := []string{"cc", "-E", "-P", "probe.c"}
		if noStandard {
			args = append(args, "-nostdinc")
		}
		result, err := CompileCommand(&CommandRequest{Filesystem: fs, Args: args, Target: "linux/amd64"})
		if err != nil {
			t.Fatal(err)
		}
		if noStandard {
			if result.Ok || result.Diagnostic.Code != "RENVO-CPP-001" {
				t.Fatalf("nostdinc: %+v", result.Diagnostic)
			}
			continue
		}
		if !result.Ok || !bytes.Contains(result.Binary, []byte("mbstate_t state")) {
			t.Fatalf("preprocess: %+v", result.Diagnostic)
		}
	}
}
