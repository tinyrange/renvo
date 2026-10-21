//go:build !renvo

package std_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"renvo.dev/internal/testfront"
)

func TestStdPackagesWithGeneratedRunner(t *testing.T) {
	root := stdRoot(t)
	packages := stdTestPackages(t, root)
	if len(packages) == 0 {
		t.Fatal("no std test packages discovered")
	}
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module renvostd.generated\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(workspace, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	var buildOnce sync.Once
	var buildOutput []byte
	var buildErr error
	for i, dir := range packages {
		name, err := filepath.Rel(root, dir)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(name, func(t *testing.T) {
			result, err := testfront.GeneratePackage(dir)
			if err != nil {
				t.Fatalf("GeneratePackage failed: %v", err)
			}
			// Keep separate globals, inputs and working directories for each
			// runner, and only generate packages selected by -run filters.
			program := fmt.Sprintf("runner_%03d", i)
			out := filepath.Join(workspace, "packages", program)
			if err := testfront.WritePackage(out, result); err != nil {
				t.Fatalf("WritePackage failed: %v", err)
			}
			executable := filepath.Join(bin, program)
			if runtime.GOOS == "windows" {
				executable += ".exe"
			}
			// Parallel children resume only after the parent has prepared
			// every selected package. Share one Go build, not test results.
			t.Parallel()
			buildOnce.Do(func() {
				build := exec.Command("go", "build", "-o", bin, "./packages/...")
				build.Dir = workspace
				build.Env = append(os.Environ(), "GOWORK=off")
				buildOutput, buildErr = build.CombinedOutput()
			})
			if buildErr != nil {
				t.Fatalf("generated runners build failed: %v\n%s", buildErr, buildOutput)
			}
			cmd := exec.Command(executable)
			cmd.Dir = out
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("generated runner failed: %v\n%s", err, output)
			}
			if !strings.Contains(string(output), "PASS") {
				t.Fatalf("generated runner output = %q", output)
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
