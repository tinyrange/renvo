package link

import (
	"os"
	"strings"
	"testing"
)

func TestCoreLinkerAlgorithmIsSharedAcrossFrontends(t *testing.T) {
	shared := readLinkSource(t, "link.go")
	if strings.Contains(shared, "//go:build") {
		t.Fatal("core linker algorithm is hidden behind a build tag")
	}
	sharedDeclarations := []string{
		"func LinkBuildCore(",
		"func linkBuildCore(",
		"func LinkUnitsCore(",
		"func LinkProgramsCore(",
		"func linkProgramsCore(",
		"func appendProgramCore(",
		"func linkedTokenActions(",
	}
	for _, declaration := range sharedDeclarations {
		if !strings.Contains(shared, declaration) {
			t.Errorf("shared linker is missing %s", declaration)
		}
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if source := readLinkSource(t, name); strings.Contains(source, "//go:build") {
			t.Errorf("production linker file %s is build-tagged", name)
		}
	}
}

func readLinkSource(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
