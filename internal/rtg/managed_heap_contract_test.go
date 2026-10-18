package rtg

import (
	"os"
	"strings"
	"testing"
)

func TestManagedHeapContractValidation(t *testing.T) {
	path := "../../backend/definitions/linux_amd64.rtg"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, old, replacement string }{
		{"missing-abi", "managed_heap_abi = 1", ""},
		{"unknown-abi", "managed_heap_abi = 1", "managed_heap_abi = 2"},
		{"word-mismatch", "managed_word_bits = 64", "managed_word_bits = 32"},
		{"missing-source", "managed_source = go managed_linux_amd64.linuxAmd64ManagedSource", ""},
		{"missing-startup", "managed_startup = go managed_linux_amd64.linuxAmd64ManagedStartup", ""},
		{"missing-boundary", "managed_boundary = go managed_linux_amd64.linuxAmd64ManagedBoundary", ""},
		{"wrong-source-signature", "managed_source = go managed_linux_amd64.linuxAmd64ManagedSource", "managed_source = go linuxObjectStaticCall"},
		{"wrong-startup-signature", "managed_startup = go managed_linux_amd64.linuxAmd64ManagedStartup", "managed_startup = go managed_linux_amd64.linuxAmd64ManagedBoundary"},
		{"wrong-boundary-signature", "managed_boundary = go managed_linux_amd64.linuxAmd64ManagedBoundary", "managed_boundary = go managed_linux_amd64.linuxAmd64ManagedStartup"},
	} {
		t.Run(test.name, func(t *testing.T) {
			text := strings.Replace(string(data), test.old, test.replacement, 1)
			if result := Resolve(ParseImports([]byte(text), path, testFilesystemImportLoader{})); result.Ok {
				t.Fatal("accepted invalid managed contract")
			}
		})
	}
	text := strings.Replace(string(data), "memory_strategy = desktop", "memory_strategy = arena", 1)
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "managed_") {
			lines = append(lines, line)
		}
	}
	if result := Resolve(ParseImports([]byte(strings.Join(lines, "\n")), path, testFilesystemImportLoader{})); !result.Ok {
		t.Fatalf("arena requires no managed hooks: %#v", result.Diagnostics)
	}
}
