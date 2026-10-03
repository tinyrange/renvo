//go:build !renvo

package rtg

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPreparedOpenFlagsAreDeclarationBound(t *testing.T) {
	const filename = "../../backend/definitions/darwin_aarch64.rtg"
	source, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, policy, create, truncate string
		fail                           bool
	}{
		{name: "renamed", policy: "open_flag_layout = bsd", create: "512", truncate: "1024"},
		{name: "omitted", create: "64", truncate: "512"},
		{name: "portable", policy: "open_flag_layout = portable", create: "64", truncate: "512"},
		{name: "unknown", policy: "open_flag_layout = magic", fail: true},
		{name: "duplicate", policy: "open_flag_layout = bsd\nopen_flag_layout = portable", fail: true},
		{name: "block", policy: "open_flag_layout { value = bsd }", fail: true},
		{name: "extra", policy: "open_flag_layout = bsd portable", fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			modified := bytes.Replace(source, []byte("open_flag_layout = bsd"), []byte(tc.policy), 1)
			// Unfamiliar public names must keep the declared bits. Conversely,
			// familiar Darwin names with no declaration cannot infer BSD bits.
			targetName := "darwin/arm64"
			if tc.name == "renamed" {
				modified = bytes.ReplaceAll(modified, []byte("darwin_arm64"), []byte("private_runtime"))
				modified = bytes.Replace(modified, []byte("target darwin/arm64 {"), []byte("target private/flags {"), 1)
				modified = bytes.Replace(modified, []byte("os = darwin"), []byte("os = private_environment"), 1)
				targetName = "private/flags"
			}
			resolved := Resolve(ParseImports(modified, filename, testFilesystemImportLoader{}))
			generated := GeneratePreparedBackend(resolved, targetName)
			if tc.fail {
				if generated.Ok || len(generated.Source) != 0 {
					t.Fatal("invalid flag layout generated code")
				}
				for _, diagnostic := range generated.Diagnostics {
					if diagnostic.Code == "RTG-VALIDATE-134" || tc.name == "duplicate" && diagnostic.Code == "RTG-VALIDATE-060" {
						return
					}
				}
				t.Fatalf("missing layout diagnostic: %#v", generated.Diagnostics)
			}
			if !generated.Ok {
				t.Fatalf("generate: %#v", generated.Diagnostics)
			}
			for _, fact := range []string{"const renvoRTGOpenCreate = " + tc.create, "const renvoRTGOpenTruncate = " + tc.truncate} {
				if !strings.Contains(string(generated.Source), fact) {
					t.Fatalf("missing %s", fact)
				}
			}
		})
	}
}
