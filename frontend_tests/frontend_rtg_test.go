package frontend_tests

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDefinitionCompilerSelfHosts(t *testing.T) {
	root := repoRoot(t)
	frontend := selfHostedFrontendCompiler(t, root)
	output := filepath.Join(t.TempDir(), "rtg-selfhost")
	command := frontendCommand(frontend,
		"-t", frontend.target, "-s", "-o", output,
		"./frontend_tests/testdata/rtg_selfhost")
	command.Dir = root
	command.Env = frontendCommandEnv(frontend.env, root)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile definition compiler with Renvo: %v\n%s", err, combined)
	}
	combined, err := exec.Command(output).CombinedOutput()
	if err != nil {
		t.Fatalf("run self-hosted definition compiler: %v\n%s", err, combined)
	}
	if string(combined) != "PASS\n" {
		t.Fatalf("self-hosted definition compiler output = %q", combined)
	}
}
