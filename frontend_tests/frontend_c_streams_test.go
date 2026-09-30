package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// Exercise the same libc stream contract on a native kernel that the trex
// build-environment integration executes in its Linux interpreter.
func TestFrontendCFileStreams(t *testing.T) {
	nativeLibcFixture(t, "streams", nil, "PASS\n")
}
func TestFrontendCWideAndExit(t *testing.T) {
	for _, arg := range []string{"", "exit", "immediate"} {
		t.Run(arg, func(t *testing.T) {
			want := "wide: Aé😀\n"
			if arg != "immediate" {
				want += "second\nfirst\n"
			}
			var args []string
			if arg != "" {
				args = []string{arg}
			}
			nativeLibcFixture(t, "wide_exit", args, want)
		})
	}
}
func nativeLibcFixture(t *testing.T, fixture string, args []string, want string) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("Linux/amd64 libc acceptance")
	}
	root := repoRoot(t)
	frontend := frontendCompiler(t, root)
	if frontend.compiler == "" {
		t.Skip("frontend compiler unavailable")
	}
	dir := t.TempDir()
	source, err := os.ReadFile(filepath.Join(root, "libc", "tests", fixture+".c"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.c"), source, 0644); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(dir, "streams")
	command := frontendCommand(frontend, "cc", "-t", "linux/amd64", "main.c", "-o", executable)
	command.Dir = dir
	command.Env = cExecutableFrontendEnv(frontend, root, dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, output)
	}
	command = exec.Command(executable, args...)
	command.Dir = dir
	command.Env = []string{"LC_ALL=C.UTF-8", "RENVO_LIBC_TEST=guest"}
	if output, err := command.CombinedOutput(); err != nil || string(output) != want {
		t.Fatalf("run: %v\n%s", err, output)
	}
	if fixture != "streams" {
		return
	}
	info, err := os.Stat(filepath.Join(dir, "data"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0111 != 0 {
		t.Fatalf("fopen created executable data file: %v", info.Mode())
	}
	data, err := os.ReadFile(filepath.Join(dir, "data"))
	if err != nil || len(data) != 0 {
		t.Fatalf("truncated output: %q %v", data, err)
	}
}
