//go:build !renvo

package rtg

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPreparedDirectorySyscallProtocolIsDefinitionBound(t *testing.T) {
	const filename = "../../backend/definitions/darwin_aarch64.rtg"
	source, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, layout, hook, code string
		wantDirectory            bool
	}{
		{"renamed", "directory_entries", "machDirectoryFromStack", "", true},
		{"omitted", "", "", "", false},
		{"explicit-registers", "register_words", "", "", false},
		{"unknown", "unknown", "", "RTG-VALIDATE-136", false},
		{"extra", "directory_entries extra", "machDirectoryFromStack", "RTG-VALIDATE-136", false},
		{"block", "directory_entries { unexpected = true }", "machDirectoryFromStack", "RTG-VALIDATE-136", false},
		{"duplicate", "directory_entries\nraw_syscall_layout = directory_entries", "machDirectoryFromStack", "RTG-VALIDATE-060", false},
		{"missing-adapter", "directory_entries", "", "RTG-VALIDATE-136", false},
		{"custom-site-table", "register_words\nsyscall { site_table = address_number_pairs }", "machDirectoryFromStack", "RTG-VALIDATE-136", false},
		{"wrong-signature", "directory_entries", "machStaticCall", "RTG-VALIDATE-032", false},
		{"unknown-adapter", "directory_entries", "missing", "RTG-VALIDATE-032", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			modified := bytes.ReplaceAll(source, []byte("darwin_arm64"), []byte("private_runtime"))
			modified = bytes.Replace(modified, []byte("target darwin/arm64 {"), []byte("target custom/directory {"), 1)
			if tc.wantDirectory {
				modified = bytes.Replace(modified, []byte("os = darwin"), []byte("os = private_environment"), 1)
			}
			layout := ""
			if tc.layout != "" {
				layout = "raw_syscall_layout = " + tc.layout
			}
			modified = bytes.Replace(modified, []byte("raw_syscall_layout = directory_entries"), []byte(layout), 1)
			hook := ""
			if tc.hook != "" {
				hook = "emit_syscall_from_stack = go " + tc.hook
			}
			modified = bytes.Replace(modified, []byte("emit_syscall_from_stack = go machDirectoryFromStack"), []byte(hook), 1)
			resolved := Resolve(ParseImports(modified, filename, testFilesystemImportLoader{}))
			generated := GeneratePreparedBackend(resolved, "custom/directory")
			if tc.code != "" {
				if generated.Ok || len(generated.Source) != 0 || !hasDiagnosticCode(generated.Diagnostics, tc.code) {
					t.Fatalf("invalid declaration generated=%v diagnostics=%#v", generated.Ok, generated.Diagnostics)
				}
				return
			}
			if !generated.Ok {
				t.Fatalf("generation failed: %#v", generated.Diagnostics)
			}
			text := string(generated.Source)
			policy := "1"
			if tc.wantDirectory {
				policy = "3"
				for _, want := range []string{
					"return rtgNativeDarwinAarch64PackageMachDirectoryFromStack(out, wordCount, number)",
					"func rtgNativeDarwinAarch64PackageMachDirectoryFromStack(",
					`"_getdirentries"`,
				} {
					if !strings.Contains(text, want) {
						t.Fatalf("missing directory adapter %q", want)
					}
				}
			}
			if !strings.Contains(text, "const renvoRTGSyscallArgumentPolicy = "+policy) {
				t.Fatalf("incorrect raw syscall argument protocol")
			}
		})
	}
}
