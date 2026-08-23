package frontend_tests

import (
	"os"
	"path/filepath"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/targetinfo"
)

func TestBrowserCProjectCompilesWithEveryAdvertisedBackend(t *testing.T) {
	root := repoRoot(t)
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "main.c"), []byte(`#include <stdio.h>
int main(void) {
    printf("C browser target PASS %d\n", 42);
    return 0;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, target := range targetinfo.All() {
		if !target.Advertised {
			continue
		}
		t.Run(target.Name, func(t *testing.T) {
			built := driver.BuildCExecutableUnitCompact(
				[]string{"main.c"}, target.Name, target.Tags,
				project, filepath.Join(root, "std"), driver.OSFS{})
			if !built.Ok {
				t.Fatalf("C browser frontend failed: %#v", built)
			}
			compiled := (backendcompiled.Backend{}).CompileUnitWithArena(built.Unit, target.Backend, true, false, 64*1024*1024)
			if !compiled.Ok || len(compiled.Binary) == 0 {
				t.Fatalf("C browser backend failed: %#v", compiled.Diagnostic)
			}
		})
	}
}
