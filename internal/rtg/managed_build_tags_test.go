package rtg

import (
	"os"
	"strings"
	"testing"
)

func TestManagedHeapDerivesRuntimeBuildTag(t *testing.T) {
	path := "../../backend/definitions/linux_amd64.rtg"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Merely opting into desktop allocation must select the safe runtime adapter,
	// even for a custom target without a manually copied internal build tag.
	text := strings.ReplaceAll(string(source), `,"renvo_managed"`, "")
	text = strings.ReplaceAll(text, `, "renvo_managed"`, "")
	text = strings.ReplaceAll(text, "linux/amd64-desktop", "custom/managed")
	for _, strategy := range []string{"desktop", "arena"} {
		resolved := Resolve(ParseImports([]byte(strings.ReplaceAll(text, "memory_strategy = desktop", "memory_strategy = "+strategy)), path, testFilesystemImportLoader{}))
		if !resolved.Ok {
			t.Fatalf("resolve: %#v", resolved.Diagnostics)
		}
		target, ok := lookupResolvedTarget(resolved, "custom/managed")
		if !ok {
			t.Fatal("missing target")
		}
		count := 0
		for _, tag := range target.Descriptor.BuildTags {
			if tag == "renvo_managed" {
				count++
			}
		}
		want := 0
		if strategy == "desktop" {
			want = 1
		}
		if count != want {
			t.Fatalf("%s: managed tags = %d, want %d", strategy, count, want)
		}
	}
}
