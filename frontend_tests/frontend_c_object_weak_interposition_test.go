package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendCObjectWeakInterpositionSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectWeakInterpositionSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectWeakInterpositionSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectWeakInterpositionSystemLink(t, selfHostedFrontendCompiler(t, root))
}

func runCObjectWeakInterpositionSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}

	dir := t.TempDir()
	files := map[string]string{
		"weak.c": `
__attribute__((weak)) int selectable(int value) { return value + 1; }
int call_selectable(int value) { return selectable(value); }
`,
		"strong.c": `int selectable(int value) { return value + 9; }`,
		"harness.c": `
#include <stdio.h>
int call_selectable(int);
int main(void) {
	if (call_selectable(4) != 13)
		return 1;
	puts("PASS");
	return 0;
}
`,
	}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"weak", "strong"} {
		command := frontendCommand(frontend, "cc", "-c", name+".c", "-o", name+".o")
		command.Dir = dir
		command.Env = frontendCommandEnv(frontend.env, dir)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("compile %s interposition object with Renvo: %v\n%s", name, err, output)
		}
	}
	executable := filepath.Join(dir, "weak-interposition-test")
	link := exec.Command(linkerDriver, "harness.c", "weak.o", "strong.o", "-o", executable)
	link.Dir = dir
	if output, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link weak interposition objects: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil || string(output) != "PASS\n" {
		t.Fatalf("run weak interposition objects: %v, output %q", err, output)
	}
}
