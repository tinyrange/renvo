package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendCObjectCallbackParameterSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectCallbackParameterSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectCallbackParameterSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectCallbackParameterSystemLink(t, selfHostedFrontendCompiler(t, root))
}

func runCObjectCallbackParameterSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "callback.c")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "callback.o")
	executable := filepath.Join(dir, "callback-test")
	if err := os.WriteFile(source, []byte(`
extern void *renvo_identity(void *);
unsigned long renvo_apply_callback(unsigned long value, void (*report)(const char *)) {
	void *raw = renvo_identity((void *)report);
	void (*converted)(const char *) = (void (*)(const char *))raw;
	if (value == 42) converted("renvo callback");
	return value + 1;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`
#include <string.h>
extern unsigned long renvo_apply_callback(unsigned long, void (*)(const char *));
static int called;
void *renvo_identity(void *value) { return value; }
static void report(const char *message) { called = strcmp(message, "renvo callback") == 0; }
int main(void) { return renvo_apply_callback(42, report) == 43 && called ? 0 : 1; }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C callback object with Renvo: %v\n%s", err, output)
	}
	link := exec.Command(linkerDriver, harness, object, "-o", executable)
	if output, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link C callback object: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked C callback object: %v, output %q", err, output)
	}
}
