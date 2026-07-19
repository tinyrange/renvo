package frontend_tests

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFrontendOSProcessSemantics(t *testing.T) {
	root := repoRoot(t)
	runFrontendOSProcessSemantics(t, frontendCompiler(t, root))
}

func TestFrontendStage3OSProcessSemantics(t *testing.T) {
	if os.Getenv(selfHostTestsEnv) != "1" {
		t.Skipf("set %s=1 to run self-hosted frontend process semantics", selfHostTestsEnv)
	}
	root := repoRoot(t)
	runFrontendOSProcessSemantics(t, selfHostedFrontendCompiler(t, root))
}

func runFrontendOSProcessSemantics(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if frontend.compiler == "" {
		t.Skip("frontend compiler unavailable")
	}
	project := writeOSProcessProject(t)
	host := filepath.Join(t.TempDir(), "host-app")
	cmd := exec.Command("go", "build", "-o", host, ".")
	cmd.Dir = project
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("host process fixture build failed: %v\n%s", err, string(out))
	}
	compiled := filepath.Join(t.TempDir(), "renvo-app")
	cmd = exec.Command(frontend.compiler, "-t", frontend.target, "-s", "-o", compiled, ".")
	cmd.Dir = project
	cmd.Env = frontendCommandEnv(frontend.env, project)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("frontend process fixture build failed: %v\n%s", err, string(out))
	}

	hostOut, hostCode := runOSProcessFixture(t, host, project, "check")
	frontOut, frontCode := runOSProcessFixture(t, compiled, project, "check")
	if hostCode != 0 || !bytes.Equal(hostOut, []byte("PASS\n")) {
		t.Fatalf("host process fixture = exit %d, output %q", hostCode, string(hostOut))
	}
	if frontCode != hostCode || !bytes.Equal(frontOut, hostOut) {
		t.Fatalf("frontend process fixture = exit %d, output %q; host = exit %d, output %q", frontCode, string(frontOut), hostCode, string(hostOut))
	}

	hostOut, hostCode = runOSProcessFixture(t, host, project, "exit")
	frontOut, frontCode = runOSProcessFixture(t, compiled, project, "exit")
	if hostCode != 23 || len(hostOut) != 0 {
		t.Fatalf("host exit fixture = exit %d, output %q", hostCode, string(hostOut))
	}
	if frontCode != hostCode || !bytes.Equal(frontOut, hostOut) {
		t.Fatalf("frontend exit fixture = exit %d, output %q; host = exit %d, output %q", frontCode, string(frontOut), hostCode, string(hostOut))
	}
}

func runOSProcessFixture(t *testing.T, executable string, dir string, mode string) ([]byte, int) {
	t.Helper()
	cmd := exec.Command(executable, mode)
	cmd.Dir = dir
	cmd.Env = []string{"PWD=" + dir, "RENVO_OS_MARKER=present"}
	out, err := cmd.CombinedOutput()
	if err == nil {
		return out, 0
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("run %s failed without exit status: %v\n%s", executable, err, string(out))
	}
	return out, exitErr.ExitCode()
}

func writeOSProcessProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/osprocess\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := `package main

import "os"

func envValue(items []string, key string) string {
	for i := 0; i < len(items); i++ {
		item := items[i]
		if len(item) > len(key) && item[:len(key)] == key && item[len(key)] == '=' {
			return item[len(key)+1:]
		}
	}
	return ""
}

func fail(message string) {
	print("FAIL ")
	print(message)
	print("\n")
}

func main() {
	if len(os.Args) != 2 {
		fail("args")
		return
	}
	if os.Args[1] == "exit" {
		os.Exit(23)
		fail("exit returned")
		return
	}
	if envValue(os.Environ(), "RENVO_OS_MARKER") != "present" {
		fail("environment")
		return
	}
	wd, err := os.Getwd()
	if err != nil || wd == "" {
		fail("working directory")
		return
	}
	if _, err := os.ReadFile("definitely-missing"); err == nil {
		fail("missing file")
		return
	}
	if err := os.WriteFile(".", []byte("x"), 0600); err == nil {
		fail("permission")
		return
	}
	if err := os.WriteFile("process.tmp", []byte("ok"), 0600); err != nil {
		fail("write")
		return
	}
	file, err := os.Open("process.tmp")
	if err != nil {
		fail("open")
		return
	}
	buf := make([]byte, 8)
	n, err := file.Read(buf)
	if n != 2 || err != nil || string(buf[:n]) != "ok" {
		fail("short read")
		return
	}
	n, err = file.Read(buf)
	if n != 0 || err == nil {
		fail("EOF")
		return
	}
	if err := file.Close(); err != nil {
		fail("close")
		return
	}
	if err := file.Close(); err == nil {
		fail("close error")
		return
	}
	print("PASS\n")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(source), 0o644); err != nil {
		t.Fatal(fmt.Errorf("write process fixture: %w", err))
	}
	return dir
}
