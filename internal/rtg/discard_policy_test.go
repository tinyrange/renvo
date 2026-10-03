//go:build !renvo

package rtg

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPreparedDiscardPolicyIsDefinitionBound(t *testing.T) {
	const filename = "../../backend/definitions/linux_amd64.rtg"
	source, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	const original = "discard_pages {\n\t\tpage_size = 4096\n\t\tnumber = 28\n\t\tadvice = 4\n\t}"
	if !bytes.Contains(source, []byte(original)) {
		t.Fatal("missing policy fixture")
	}
	for _, tc := range []struct {
		name, policy, os, page, number, advice string
		fail, sites                            bool
	}{
		{name: "renamed", policy: original, os: "private_environment", page: "4096", number: "28", advice: "4"},
		{name: "absent", os: "linux", page: "0", number: "0", advice: "0"},
		{name: "independent", policy: "discard_pages { page_size = 8192; number = 93; advice = 7 }", os: "private_environment", page: "8192", number: "93", advice: "7", sites: true},
		{name: "zero", policy: "discard_pages { page_size = 0; number = 28; advice = 4 }", os: "linux", fail: true},
		{name: "nonpower", policy: "discard_pages { page_size = 6000; number = 28; advice = 4 }", os: "linux", fail: true},
		{name: "missing", policy: "discard_pages { page_size = 4096; number = 28 }", os: "linux", fail: true},
		{name: "unknown", policy: "discard_pages { page_size = 4096; number = 28; advice = 4; mystery = 1 }", os: "linux", fail: true},
		{name: "duplicate", policy: original + "\n" + original, os: "linux", fail: true},
		{name: "duplicate-field", policy: "discard_pages { page_size = 4096; number = 28; advice = 4; advice = 4 }", os: "linux", fail: true},
		{name: "negative", policy: "discard_pages { page_size = 4096; number = -1; advice = 4 }", os: "linux", fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			modified := bytes.Replace(source, []byte(original), []byte(strings.ReplaceAll(tc.policy, "; ", "\n")), 1)
			modified = bytes.Replace(modified, []byte("target linux/amd64 {"), []byte("target private/runtime {"), 1)
			modified = bytes.Replace(modified, []byte("os = linux"), []byte("os = "+tc.os), 1)
			if tc.sites {
				modified = bytes.Replace(modified, []byte("syscall {"), []byte("syscall {\nsite_table = address_number_pairs"), 1)
			}
			resolved := Resolve(ParseImports(modified, filename, testFilesystemImportLoader{}))
			if !resolved.Ok {
				t.Fatalf("resolve fixture: %#v", resolved.Diagnostics)
			}
			generated := GeneratePreparedBackend(resolved, "private/runtime")
			if tc.fail {
				if generated.Ok || len(generated.Source) != 0 {
					t.Fatal("invalid discard policy generated output")
				}
				for _, diagnostic := range generated.Diagnostics {
					if diagnostic.Code == "RTG-VALIDATE-131" {
						return
					}
				}
				t.Fatalf("missing discard-policy diagnostic: %#v", generated.Diagnostics)
			}
			if !generated.Ok {
				t.Fatalf("generate fixture: %#v", generated.Diagnostics)
			}
			text := string(generated.Source)
			for _, want := range []string{
				"const renvoRTGDiscardPageSize = " + tc.page,
				"const renvoRTGDiscardNumber = " + tc.number,
				"const renvoRTGDiscardAdvice = " + tc.advice,
				"return renvoRTGDiscardPageSize != 0",
				"renvoRTGSyscallWord2, renvoRTGDiscardAdvice",
				"renvoRTGSyscallNumber, renvoRTGDiscardNumber",
			} {
				if !strings.Contains(text, want) {
					t.Errorf("missing %q", want)
				}
			}
			if strings.Contains(text, "a.openbsdSyscalls = append(a.openbsdSyscalls, renvoRTGDiscardNumber)") != tc.sites {
				t.Fatal("discard syscall-site metadata does not follow declaration")
			}
		})
	}
}
