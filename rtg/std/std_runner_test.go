//go:build !rtg

package std_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"j5.nz/rtg/rtg/internal/testfront"
)

func TestStdPackagesWithGeneratedRunner(t *testing.T) {
	root := stdRoot(t)
	packages := stdTestPackages(t, root)
	if len(packages) == 0 {
		t.Fatalf("no std test packages discovered")
	}
	for _, dir := range packages {
		dir := dir
		name, err := filepath.Rel(root, dir)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(name, func(t *testing.T) {
			result, err := testfront.GeneratePackage(dir)
			if err != nil {
				t.Fatalf("GeneratePackage failed: %v", err)
			}
			out := t.TempDir()
			if err := testfront.WritePackage(out, result); err != nil {
				t.Fatalf("WritePackage failed: %v", err)
			}
			if err := os.WriteFile(filepath.Join(out, "go.mod"), []byte("module rtgstd.generated\n\ngo 1.25\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("go", "run", ".")
			cmd.Dir = out
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("generated runner failed: %v\n%s", err, string(output))
			}
			if !strings.Contains(string(output), "PASS") {
				t.Fatalf("generated runner output = %q", string(output))
			}
		})
	}
}

func stdRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}

func stdTestPackages(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() || path == root {
			return nil
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, item := range entries {
			if !item.IsDir() && strings.HasSuffix(item.Name(), "_test.go") {
				out = append(out, path)
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
