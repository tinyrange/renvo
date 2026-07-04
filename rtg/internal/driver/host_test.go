//go:build !rtg

package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func TestRunCommandWithRealBackendCompilesMainEntrypoint(t *testing.T) {
	runHostMainCase(t, []byte(`package main

func calc(a int, b int, c int) int {
	total := (a + b) * c
	total = total - b
	total = total + a%5
	return total
}

func main() {
	if calc(3, 5, 2) == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`))
}

func TestRunCommandWithRealBackendCompilesStructMethods(t *testing.T) {
	runHostMainCase(t, []byte(`package main

type pair struct {
	a int
	b int
}

func (p pair) score() int {
	return p.a*10 + p.b
}

func (p *pair) add(v int) {
	p.b = p.b + v
}

func main() {
	p := pair{a: 1, b: 2}
	p.add(3)
	if p.score() == 15 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`))
}

func TestRunCommandWithRealBackendCompilesArrayLiterals(t *testing.T) {
	runHostMainCase(t, []byte(`package main

func main() {
	grid := [2][3]int{{1, 0, 3}, {4, 5, 2}}
	if grid[0][1]+grid[1][2] == 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`))
}

func runHostMainCase(t *testing.T, src []byte) {
	t.Helper()
	target, ok := hostRunnableTarget()
	if !ok {
		t.Skipf("no directly runnable frontend target for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	repoRoot := driverRepoRoot(t)
	dir := writeHostSourceCase(t, src)
	backend := filepath.Join(t.TempDir(), "rtgx-backend")
	cmd := exec.Command("go", "build", "-o", backend, ".")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("backend build failed: %v\n%s", err, string(out))
	}

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

	result := RunCommand(
		[]string{"rtg", "-t", target, "-s", "-o", "app", "./cmd/app"},
		[]string{BackendEnv + "=" + backend},
		nil,
	)
	if !result.Ok {
		t.Fatalf("RunCommand failed: err=%d path=%q compile=%#v", result.Error, result.ErrorPath, result.Compile)
	}
	out, err := exec.Command(filepath.Join(dir, "app")).CombinedOutput()
	if err != nil {
		t.Fatalf("compiled app failed: %v\n%s", err, string(out))
	}
	if string(out) != "PASS\n" {
		t.Fatalf("compiled app output = %q", string(out))
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

func writeHostSourceCase(t *testing.T, src []byte) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/case\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	appDir := filepath.Join(dir, "cmd", "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.go"), src, 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	return dir
}

func driverRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func hostRunnableTarget() (string, bool) {
	if runtime.GOOS != "linux" {
		return "", false
	}
	if runtime.GOARCH == "amd64" {
		return "linux/amd64", true
	}
	if runtime.GOARCH == "386" {
		return "linux/386", true
	}
	if runtime.GOARCH == "arm" {
		return "linux/arm", true
	}
	if runtime.GOARCH == "arm64" {
		return "linux/aarch64", true
	}
	return "", false
}
