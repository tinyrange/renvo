//go:build !renvo

package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRunScriptCommandCompilesImageAndForwardsArguments(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no native Renvo target for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	repoRoot := driverRepoRoot(t)
	backend := filepath.Join(t.TempDir(), "renvo-backend")
	if runtime.GOOS == "windows" {
		backend += ".exe"
	}
	command := exec.Command("go", "build", "-o", backend, "./backend")
	command.Dir = repoRoot
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("backend build failed: %v\n%s", err, output)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/script\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	script := []byte(`import "os"
if len(os.Args) == 2 { os.WriteFile(os.Args[1], []byte("PASS\n"), 0644) }
`)
	if err := os.WriteFile(filepath.Join(dir, "hello.go"), script, 0o644); err != nil {
		t.Fatal(err)
	}
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldDir) }()
	env := []string{
		StdRootEnv + "=" + filepath.Join(repoRoot, "std"),
	}
	marker := filepath.Join(dir, "result.txt")
	result := RunScriptCommand(
		[]string{"renvo", "run", "hello.go", "--", marker},
		env, CommandBackend{Path: backend}, os.Stdin, os.Stdout, os.Stderr,
	)
	if !result.Ok || result.ExitCode != 0 {
		t.Fatalf("run failed: %#v", result)
	}
	output, err := os.ReadFile(marker)
	if err != nil || string(output) != "PASS\n" {
		t.Fatalf("script marker = %q, %v", output, err)
	}
	if result.Loader != "jit" {
		t.Fatalf("loader = %q, want jit", result.Loader)
	}
}
