package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendCObjectCleanupSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectCleanupSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectCleanupSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectCleanupSystemLink(t, selfHostedFrontendCompiler(t, root))
}

func runCObjectCleanupSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "cleanup.c")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "cleanup.o")
	executable := filepath.Join(dir, "cleanup-test")
	if err := os.WriteFile(source, []byte(`
static int trace;
static void record(int *value) { trace = trace * 10 + *value; }
int run_cleanup(void) {
	trace = 0;
	{
		int outer __attribute__((cleanup(record))) = 2;
		{
			int inner __attribute__((cleanup(record))) = 3;
		}
	}
	return trace;
}
int cleanup_trace(void) { return trace; }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`
#include <stdio.h>
extern int run_cleanup(void);
extern int cleanup_trace(void);
int main(void) {
	run_cleanup();
	int first = cleanup_trace();
	run_cleanup();
	int second = cleanup_trace();
	if (first == 32 && second == 32) return 0;
	printf("cleanup results: %d %d\n", first, second);
	return 1;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C cleanup object with Renvo: %v\n%s", err, output)
	}
	link := exec.Command(linkerDriver, harness, object, "-o", executable)
	if output, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link C cleanup object: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked C cleanup object: %v, output %q", err, output)
	}
}
