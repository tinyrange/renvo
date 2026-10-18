package frontend_tests

import (
	"os"
	"path/filepath"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/targetinfo"
)

func TestBrowserCProjectPolicyForEveryAdvertisedBackend(t *testing.T) {
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
			if targetinfo.MemoryStrategy(target.Name) == "desktop" {
				// Managed profiles currently support Go process executables only.
				// Exercise rejection, rather than silently omitting these targets.
				if compiled.Ok || len(compiled.Binary) != 0 || compiled.Diagnostic.Phase != "backend" || compiled.Diagnostic.Code != "RENVO-BACKEND-001" {
					t.Fatalf("managed C compilation should fail without an artifact: %#v", compiled)
				}
				// A broken backend must not satisfy the negative C assertion.
				goBinary, goOK := backendcompiled.RenvoCompileSourceToBytesWithOptions(
					[]byte("package main; func appMain() int { return 0 }"), target.Backend,
					backendcompiled.RenvoCompileOptions{ArenaSize: 64 * 1024 * 1024})
				if !goOK || len(goBinary) == 0 {
					t.Fatalf("managed Go executable failed: success=%v, bytes=%d", goOK, len(goBinary))
				}
				return
			}
			if !compiled.Ok || len(compiled.Binary) == 0 {
				t.Fatalf("C browser backend failed: %#v", compiled.Diagnostic)
			}
		})
	}
}
