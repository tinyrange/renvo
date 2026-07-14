package rtg_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrontendStage3DiagnosticSurvivesArenaReset(t *testing.T) {
	if os.Getenv(selfHostTestsEnv) != "1" {
		t.Skipf("set %s=1 to run self-hosted frontend diagnostics", selfHostTestsEnv)
	}
	root := repoRoot(t)
	frontend := selfHostedFrontendCompiler(t, root)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/diagnostic\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	appDir := filepath.Join(dir, "cmd", "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := "package main\n\nimport _ \"github.com/example/missing\"\n\nfunc main() {}\n"
	if err := os.WriteFile(filepath.Join(appDir, "main.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "app")
	cmd := exec.Command(frontend.compiler, "-t", frontend.target, "-s", "-o", output, "./cmd/app")
	cmd.Dir = dir
	cmd.Env = frontendCommandEnv(frontend.env, dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("stage3 unexpectedly accepted an unresolved external import")
	}
	want := "rtg: unresolved import: github.com/example/missing"
	if !strings.Contains(string(out), want) {
		t.Fatalf("stage3 diagnostic = %q, want it to contain %q", string(out), want)
	}
}
