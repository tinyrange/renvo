package rtg

import (
	"os"
	"strings"
	"testing"
)

func TestManagedHeapTargetPolicy(t *testing.T) {
	path := "../../backend/definitions/linux_amd64.rtg"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		strategy string
		valid    bool
	}{{"desktop", true}, {"arena", true}, {"automatic", false}} {
		text := strings.Replace(string(source), "memory_strategy = desktop", "memory_strategy = "+test.strategy, 1)
		result := Resolve(ParseImports([]byte(text), path, testFilesystemImportLoader{}))
		if result.Ok != test.valid {
			t.Fatalf("%s: %#v", test.strategy, result.Diagnostics)
		}
		if result.Ok {
			target, ok := lookupResolvedTarget(result, "linux/amd64-desktop")
			if !ok {
				t.Fatal("target missing")
			}
			want := ""
			if test.strategy == "desktop" {
				want = "desktop"
			}
			if target.Descriptor.MemoryStrategy != want {
				t.Fatalf("strategy: %q", target.Descriptor.MemoryStrategy)
			}
		}
	}
	text := strings.ReplaceAll(string(source), "managed_heap_abi = 1", "managed_heap_abi = 99")
	if result := Resolve(ParseImports([]byte(text), path, testFilesystemImportLoader{})); result.Ok {
		t.Fatal("accepted unsupported managed ABI")
	}
}
