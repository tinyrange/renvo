//go:build !renvo

package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunTestCommandCompilesAndRunsGoTests(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no native Renvo target for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	repoRoot := driverRepoRoot(t)
	backend := filepath.Join(t.TempDir(), "renvo-backend")
	command := exec.Command("go", "build", "-o", backend, "./backend")
	command.Dir = repoRoot
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("backend build failed: %v\n%s", err, output)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/renvotest\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "value.go"), []byte("package value\nfunc Add(a, b int) int { return a + b }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testPath := filepath.Join(dir, "value_test.go")
	if err := os.WriteFile(testPath, []byte("package value\nimport \"testing\"\nfunc TestAdd(t *testing.T) { if Add(2, 3) != 5 { t.Fatal(\"bad sum\") } }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := []string{
		BackendEnv + "=" + backend,
		StdRootEnv + "=" + filepath.Join(repoRoot, "std"),
	}
	result := RunTestCommand([]string{"renvo", "test", dir}, env, nil, os.Stdin, os.Stdout, os.Stderr)
	if !result.Ok || result.ExitCode != 0 {
		t.Fatalf("passing test result = %#v", result)
	}
	assertNoTemporaryTestFiles(t, dir)

	if err := os.WriteFile(testPath, []byte("package value\nimport \"testing\"\nfunc TestAdd(t *testing.T) { t.Fatal(\"broken\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result = RunTestCommand([]string{"renvo", "test", dir}, env, nil, os.Stdin, os.Stdout, os.Stderr)
	if !result.Ok || result.ExitCode == 0 {
		t.Fatalf("failing test result = %#v", result)
	}
	assertNoTemporaryTestFiles(t, dir)
}

func assertNoTemporaryTestFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "zz_renvotest_") {
			t.Fatalf("temporary generated file was not removed: %s", entry.Name())
		}
	}
}

func TestParseTestCommand(t *testing.T) {
	compile, dir, err, arg := parseTestCommand([]string{"renvo", "test", "-s", "-tags", "integration", "./widget"})
	if err != TestOK || arg != "" || dir != "./widget" {
		t.Fatalf("parse result = %#v, %q, %d, %q", compile, dir, err, arg)
	}
	if strings.Join(compile, " ") != "-s -tags integration" {
		t.Fatalf("compile args = %#v", compile)
	}
	_, _, err, arg = parseTestCommand([]string{"renvo", "test", "./one", "./two"})
	if err != TestErrArguments || arg != "./two" {
		t.Fatalf("extra package = %d, %q", err, arg)
	}
}
