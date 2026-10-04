//go:build !renvo

package rtg

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// Exercise complete prepared generation: display identities must neither add
// nor remove the syscall instruction/number metadata required by the image.
func TestPreparedSyscallSiteTableIsDefinitionBound(t *testing.T) {
	const filename = "../../backend/definitions/openbsd_amd64.rtg"
	source, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, os, table string
		wantSites       bool
		wantError       bool
	}{
		{"renamed", "private_environment", "address_number_pairs", true, false},
		{"absent", "openbsd", "", false, false},
		{"unknown", "openbsd", "unknown_table", false, true},
		{"list", "openbsd", "[address_number_pairs]", false, true},
		{"duplicate", "openbsd", "address_number_pairs\nsite_table = address_number_pairs", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			modified := bytes.Replace(source, []byte("target openbsd/amd64 {"), []byte("target custom/runtime {"), 1)
			modified = bytes.Replace(modified, []byte("os = openbsd"), []byte("os = "+tc.os), 1)
			field := ""
			if tc.table != "" {
				field = "site_table = " + tc.table
			}
			modified = bytes.Replace(modified, []byte("site_table = address_number_pairs"), []byte(field), 1)
			resolved := Resolve(ParseImports(modified, filename, testFilesystemImportLoader{}))
			if !resolved.Ok {
				t.Fatalf("resolve fixture: %#v", resolved.Diagnostics)
			}
			generated := GeneratePreparedBackend(resolved, "custom/runtime")
			if tc.wantError {
				if generated.Ok || len(generated.Source) != 0 {
					t.Fatal("unknown syscall-site layout generated output")
				}
				for _, diagnostic := range generated.Diagnostics {
					if diagnostic.Code == "RTG-VALIDATE-130" {
						return
					}
				}
				t.Fatalf("missing site-table diagnostic: %#v", generated.Diagnostics)
			}
			if !generated.Ok {
				t.Fatalf("generate fixture: %#v", generated.Diagnostics)
			}
			text := string(generated.Source)
			policy := "1"
			if tc.wantSites {
				policy = "2"
			}
			if !strings.Contains(text, "const renvoRTGSyscallArgumentPolicy = "+policy) {
				t.Fatal("raw syscall constant-number policy does not follow the site table")
			}
			if strings.Contains(text, "out.openbsdSyscalls = append(out.openbsdSyscalls, number)") != tc.wantSites {
				t.Fatal("raw syscall adapter does not preserve required site metadata")
			}
			start := strings.Index(text, "func renvoRTGEmitRuntimeOperation(")
			if start < 0 {
				t.Fatal("missing runtime operation adapter")
			}
			body := text[start:]
			if end := strings.Index(body, "\n}\n"); end >= 0 {
				body = body[:end]
			} else {
				t.Fatal("unterminated runtime operation adapter")
			}
			wantCount := 0
			if tc.wantSites {
				wantCount = 7
			}
			if got := strings.Count(body, "out.openbsdSyscalls = append(out.openbsdSyscalls, len(out.code))"); got != wantCount {
				t.Fatalf("syscall site count = %d, want %d", got, wantCount)
			}
			for _, number := range []string{"3", "4", "169", "170", "5", "6", "124"} {
				entry := "out.openbsdSyscalls = append(out.openbsdSyscalls, " + number + ")"
				if strings.Contains(body, entry) != tc.wantSites {
					t.Fatalf("incorrect number recording for %s", number)
				}
			}
		})
	}
}
