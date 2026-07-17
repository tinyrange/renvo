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

	for _, name := range []string{"link_full.go", "link_rtg.go"} {
		source := readLinkSource(t, name)
		for _, declaration := range sharedDeclarations {
			if strings.Contains(source, declaration) {
				t.Errorf("%s redeclares shared linker algorithm %s", name, declaration)
			}
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
