package rtg_tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrontendCoreAlgorithmsAreSharedAcrossBuilds(t *testing.T) {
	root := repoRoot(t)
	shared := []struct {
		path        string
		declaration string
	}{
		{"rtg/internal/check/core.go", "func CheckGraphHeadersCore("},
		{"rtg/internal/build/core.go", "func buildProgramsCore("},
		{"rtg/internal/lower/unit.go", "func EmitCheckedPackageCore("},
		{"rtg/internal/unit/core_marshal.go", "func MarshalCore("},
	}
	for _, item := range shared {
		source := readFrontendCoreSource(t, root, item.path)
		if strings.Contains(source, "//go:build") {
			t.Errorf("%s hides the shared algorithm behind a build tag", item.path)
		}
		if !strings.Contains(source, item.declaration) {
			t.Errorf("%s is missing %s", item.path, item.declaration)
		}
	}

	adapters := []string{
		"rtg/internal/build/build.go",
		"rtg/internal/build/build_full.go",
		"rtg/internal/lower/unit_rtg.go",
		"rtg/internal/lower/unit_full.go",
		"rtg/internal/unit/unit.go",
		"rtg/internal/unit/unit_full.go",
	}
	for _, path := range adapters {
		source := readFrontendCoreSource(t, root, path)
		for _, item := range shared {
			if strings.Contains(source, item.declaration) {
				t.Errorf("%s redeclares shared algorithm %s", path, item.declaration)
			}
		}
	}
}

func readFrontendCoreSource(t *testing.T, root string, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
