package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFrontendCMixedFloatArithmetic(t *testing.T) {
	root := repoRoot(t)
	frontend := frontendCompiler(t, root)
	dir := t.TempDir()
	source := `#include <stdio.h>
int main(void) {
    float f = 3.5f;
    double d = 3.5;
    int i = 2;
    long long wide = 2;
    unsigned long long u = 2;
    if (f/i != 1.75f || i/f < 0.57f || i/f > 0.58f) return 1;
    if (f*wide != 7.0f || wide*f != 7.0f) return 2;
    if (d/u != 1.75 || u/d < 0.57 || u/d > 0.58) return 3;
    if (f+i != 5.5f || i+f != 5.5f || f-i != 1.5f) return 4;
    if (!(f > i) || !(wide < f) || !(d > u)) return 5;
    f /= i;
    if (f != 1.75f) return 6;
    printf("PASS\n");
    return 0;
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(dir, "app")
	command := frontendCommand(frontend, "cc", "-t", frontend.target, "main.c", "-o", executable)
	command.Dir = dir
	command.Env = cExecutableFrontendEnv(frontend, root, dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil || string(output) != "PASS\n" {
		t.Fatalf("mixed floating arithmetic: %v\n%s", err, output)
	}
}
