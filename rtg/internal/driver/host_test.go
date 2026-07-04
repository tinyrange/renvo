//go:build !rtg

package driver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOSFSReadsFilesAndDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/case\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "cmd"), 0o755); err != nil {
		t.Fatalf("failed to create cmd: %v", err)
	}

	fs := OSFS{}
	data, ok := fs.ReadFile(filepath.Join(dir, "go.mod"))
	if !ok || string(data) != "module example.com/case\n" {
		t.Fatalf("ReadFile = %q/%v", string(data), ok)
	}
	entries, ok := fs.ReadDir(dir)
	if !ok {
		t.Fatal("ReadDir failed")
	}
	if len(entries) != 2 {
		t.Fatalf("entry count = %d, want 2: %#v", len(entries), entries)
	}
	foundFile := false
	foundDir := false
	for i := 0; i < len(entries); i++ {
		if entries[i].Name == "go.mod" && !entries[i].IsDir {
			foundFile = true
		}
		if entries[i].Name == "cmd" && entries[i].IsDir {
			foundDir = true
		}
	}
	if !foundFile || !foundDir {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestCompileAndWrite(t *testing.T) {
	dir := writeHostCase(t)
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatalf("restore Chdir failed: %v", err)
		}
	}()

	backend := &recordingBackend{binary: []byte("binary")}
	result := CompileAndWrite([]string{"-t", "linux/amd64", "-s", "-o", "app", "./cmd/app"}, "/std", backend)
	if !result.Ok {
		t.Fatalf("CompileAndWrite failed: err=%d path=%q compile=%#v", result.Error, result.ErrorPath, result.Compile)
	}
	if backend.target != "linux/amd64" || !backend.strip {
		t.Fatalf("backend target/strip = %q/%v", backend.target, backend.strip)
	}
	data, err := os.ReadFile(filepath.Join(dir, "app"))
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if string(data) != "binary" {
		t.Fatalf("output = %q", string(data))
	}
	info, err := os.Stat(filepath.Join(dir, "app"))
	if err != nil {
		t.Fatalf("stat output failed: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("output mode = %v, want executable bit", info.Mode().Perm())
	}
}

func TestCompileAndWriteReportsCompileFailure(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatalf("restore Chdir failed: %v", err)
		}
	}()

	backend := &recordingBackend{binary: []byte("binary")}
	result := CompileAndWrite([]string{"-o", "app", "./cmd/app"}, "/std", backend)
	if result.Ok || result.Error != HostErrCompile {
		t.Fatalf("missing source result = %#v", result)
	}
	if backend.called {
		t.Fatal("backend was called after compile failure")
	}
}

func TestRunCommandDropsProgramName(t *testing.T) {
	dir := writeHostCase(t)
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatalf("restore Chdir failed: %v", err)
		}
	}()

	backend := &recordingBackend{binary: []byte("binary")}
	result := RunCommand([]string{"rtg", "-o", "app", "./cmd/app"}, nil, backend)
	if !result.Ok {
		t.Fatalf("RunCommand failed: err=%d compile=%#v", result.Error, result.Compile)
	}
	if !backend.called {
		t.Fatal("backend was not called")
	}
	if result.Compile.Build.Options.Package != "./cmd/app" || result.Compile.Build.Options.Output != "app" {
		t.Fatalf("options = %#v", result.Compile.Build.Options)
	}
	data, err := os.ReadFile(filepath.Join(dir, "app"))
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if string(data) != "binary" {
		t.Fatalf("output = %q", string(data))
	}
}

func TestCompileAndWriteWithEnvReportsMissingBackend(t *testing.T) {
	result := CompileAndWriteWithEnv([]string{"-o", "app", "./cmd/app"}, nil, nil)
	if result.Ok || result.Error != HostErrBackend {
		t.Fatalf("missing backend result = %#v", result)
	}
}

func TestHostEnvHelpers(t *testing.T) {
	env := []string{
		"OTHER=value",
		BackendEnv + "=/tmp/backend",
		StdRootEnv + "=/tmp/std",
		BackendEnv + "_EXTRA=ignored",
	}
	backend, ok := CommandBackendFromEnv(env)
	if !ok {
		t.Fatal("CommandBackendFromEnv failed")
	}
	if backend.Path != "/tmp/backend" {
		t.Fatalf("backend path = %q", backend.Path)
	}
	if got := StdRootFromEnv(env); got != "/tmp/std" {
		t.Fatalf("std root = %q", got)
	}
	if got := StdRootFromEnv(nil); got != DefaultStdRoot {
		t.Fatalf("default std root = %q", got)
	}
	if got := EnvValue(env, BackendEnv+"_EXTRA"); got != "ignored" {
		t.Fatalf("extra env = %q", got)
	}
}

func writeHostCase(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/case\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	appDir := filepath.Join(dir, "cmd", "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.go"), []byte(`package main

func appMain() int { return 0 }
`), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	return dir
}
