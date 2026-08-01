package frontend_tests

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestFrontendSingleFileSuite is the fast, broad semantic smoke test. The
// larger corpora remain authoritative for multi-package behavior, diagnostics,
// build selection, self-hosting, target formats, and focused regressions.
func TestFrontendSingleFileSuite(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	frontend := frontendCompiler(t, root)

	host := filepath.Join(dir, "host-single-file-suite")
	cmd := exec.Command("go", "build", "-o", host, "./frontend_tests/single_file")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("host suite build failed: %v\n%s", err, string(output))
	}
	runSingleFileSuite(t, host, dir)

	if frontend.compiler == "" {
		t.Skip("frontend compiler unavailable")
	}
	compiled := filepath.Join(dir, "renvo-single-file-suite")
	cmd = frontendCommand(frontend, "-t", frontend.target, "-s", "-o", compiled, "./frontend_tests/single_file")
	cmd.Dir = root
	cmd.Env = frontendCommandEnv(frontend.env, root)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("frontend suite compile failed: %v\n%s", err, string(output))
	}
	runSingleFileSuite(t, compiled, dir)
}

func TestFrontendMicrocontrollerSingleFileSuite(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	frontend := frontendCompiler(t, root)

	host := filepath.Join(dir, "host-microcontroller-suite")
	cmd := exec.Command("go", "build", "-o", host, "./frontend_tests/single_file_microcontroller")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("host microcontroller suite build failed: %v\n%s", err, string(output))
	}
	runPrintOnlySuite(t, host, dir)

	if frontend.compiler == "" {
		t.Skip("frontend compiler unavailable")
	}
	compiled := filepath.Join(dir, "renvo-microcontroller-suite")
	cmd = frontendCommand(frontend, "-t", frontend.target, "-s", "-o", compiled, "./frontend_tests/single_file_microcontroller")
	cmd.Dir = root
	cmd.Env = frontendCommandEnv(frontend.env, root)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("frontend microcontroller suite compile failed: %v\n%s", err, string(output))
	}
	runPrintOnlySuite(t, compiled, dir)
}

func runSingleFileSuite(t *testing.T, executable string, dir string) {
	t.Helper()
	cmd := exec.Command(executable, "suite-argument")
	cmd.Dir = dir
	cmd.Env = []string{"PWD=" + dir, "RENVO_SINGLE_FILE_MARKER=present"}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("single-file suite failed: %v\n%s", err, string(output))
	}
	if !bytes.Equal(output, []byte("PASS\n")) {
		t.Fatalf("single-file suite output = %q, want PASS\\n", string(output))
	}
}

func runPrintOnlySuite(t *testing.T, executable string, dir string) {
	t.Helper()
	cmd := exec.Command(executable)
	cmd.Dir = dir
	cmd.Env = []string{"PWD=" + dir}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("microcontroller single-file suite failed: %v\n%s", err, string(output))
	}
	if !bytes.Equal(output, []byte("PASS\n")) {
		t.Fatalf("microcontroller single-file suite output = %q, want PASS\\n", string(output))
	}
}
