package frontend_tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrontendPureCExecutableWithBuiltinLibc(t *testing.T) {
	root := repoRoot(t)
	runPureCExecutableWithBuiltinLibc(t, root, frontendCompiler(t, root))
}

func TestFrontendStage3PureCExecutableWithBuiltinLibc(t *testing.T) {
	root := repoRoot(t)
	runPureCExecutableWithBuiltinLibc(t, root, selfHostedFrontendCompiler(t, root))
}

func runPureCExecutableWithBuiltinLibc(t *testing.T, root string, frontend frontendConfig) {
	t.Helper()
	if frontend.compiler == "" {
		t.Skip("frontend compiler unavailable")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "main.c")
	executable := filepath.Join(dir, "app")
	program := []byte(`#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
int main(int argc, char **argv) {
    int len = argc;
    char *word = malloc(16);
    assert(word != NULL);
    strcpy(word, "renvo");
    printf("argc=%d arg=%s word=%s float=%.2f len=%d\n", argc, argv[1], word, 2.5, len);
    free(word);
    return 0;
}
`)
	if err := os.WriteFile(source, program, 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-t", frontend.target, filepath.Base(source), "-o", executable)
	command.Dir = dir
	command.Env = cExecutableFrontendEnv(frontend, root, dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile module-free C executable: %v\n%s", err, output)
	}
	command = exec.Command(executable, "probe")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run C executable: %v\n%s", err, output)
	}
	if want := []byte("argc=2 arg=probe word=renvo float=2.50 len=2\n"); !bytes.Equal(output, want) {
		t.Fatalf("C executable output = %q, want %q", output, want)
	}
}

func TestFrontendCCModeDoesNotExposeGoBuiltins(t *testing.T) {
	root := repoRoot(t)
	runCCModeDoesNotExposeGoBuiltins(t, root, frontendCompiler(t, root))
}

func TestFrontendStage3CCModeDoesNotExposeGoBuiltins(t *testing.T) {
	root := repoRoot(t)
	runCCModeDoesNotExposeGoBuiltins(t, root, selfHostedFrontendCompiler(t, root))
}

func TestFrontendCPragmaGoExecutable(t *testing.T) {
	root := repoRoot(t)
	runCPragmaGoExecutable(t, root, frontendCompiler(t, root))
}

func TestFrontendStage3CPragmaGoExecutable(t *testing.T) {
	root := repoRoot(t)
	runCPragmaGoExecutable(t, root, selfHostedFrontendCompiler(t, root))
}

func runCPragmaGoExecutable(t *testing.T, root string, frontend frontendConfig) {
	t.Helper()
	if frontend.compiler == "" {
		t.Skip("frontend compiler unavailable")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte(`#pragma go "bridge.go"
#include <stdio.h>
extern int go_value(void);
int main(void) {
    printf("pragma go PASS %d\n", go_value());
    return 0;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bridge.go"), []byte("package main\nfunc go_value() int32 { return 42 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Explicit C mode must not absorb adjacent Go files unless named by the
	// pragma. A syntax error here makes accidental directory-wide loading fail.
	if err := os.WriteFile(filepath.Join(dir, "ignored.go"), []byte("this is not Go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(dir, "app")
	command := frontendCommand(frontend, "cc", "-t", frontend.target, "main.c", "-o", executable)
	command.Dir = dir
	command.Env = cExecutableFrontendEnv(frontend, root, dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile #pragma go executable: %v\n%s", err, output)
	}
	output, err := exec.Command(executable).CombinedOutput()
	if err != nil {
		t.Fatalf("run #pragma go executable: %v\n%s", err, output)
	}
	if want := []byte("pragma go PASS 42\n"); !bytes.Equal(output, want) {
		t.Fatalf("#pragma go output = %q, want %q", output, want)
	}
}

func runCCModeDoesNotExposeGoBuiltins(t *testing.T, root string, frontend frontendConfig) {
	t.Helper()
	if frontend.compiler == "" {
		t.Skip("frontend compiler unavailable")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "main.c")
	if err := os.WriteFile(source, []byte("int main(void) { print(\"not C\\n\"); return 0; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-t", frontend.target, filepath.Base(source), "-o", filepath.Join(dir, "app"))
	command.Dir = dir
	command.Env = cExecutableFrontendEnv(frontend, root, dir)
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("cc mode unexpectedly exposed Go's print builtin")
	}
	if !strings.Contains(string(output), "undefined identifier: print") {
		t.Fatalf("cc builtin diagnostic = %q", output)
	}
}

func cExecutableFrontendEnv(frontend frontendConfig, root string, workDir string) []string {
	extra := make([]string, 0, len(frontend.env)+1)
	extra = append(extra, frontend.env...)
	extra = append(extra, "RENVO_STDROOT="+filepath.Join(root, "std"))
	return frontendCommandEnv(extra, workDir)
}
