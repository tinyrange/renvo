package frontend_tests

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
		{"internal/syntax/parse.go", "func ParseFile("},
		{"internal/check/core.go", "func CheckGraphHeadersCore("},
		{"internal/build/core.go", "func buildProgramsCore("},
		{"internal/lower/unit.go", "func EmitCheckedPackageCore("},
		{"internal/unit/core_marshal.go", "func MarshalCore("},
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

	for _, relative := range []string{
		"internal/syntax",
		"internal/check",
		"internal/build",
		"internal/link",
		"internal/lower",
		"internal/unit",
	} {
		dir := filepath.Join(root, filepath.FromSlash(relative))
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			path := filepath.Join(relative, name)
			if source := readFrontendCoreSource(t, root, path); strings.Contains(source, "//go:build") {
				t.Errorf("frontend core implementation %s is build-tagged", path)
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
