//go:build renvo_bundle

package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"renvo.dev/driver"
	"runtime"
	"testing"
	"testing/fstest"
)

func TestFrontendCHeaderSemantics(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("Linux/amd64 ABI")
	}
	source, err := os.ReadFile(filepath.Join(repoRoot(t), "backend", "tests", "c_header_semantics.c"))
	if err != nil {
		t.Fatal(err)
	}
	files := linkSourceFS{fstest.MapFS{"main.c": {Data: source}}}
	result, err := driver.CompileCommand(&driver.CommandRequest{Filesystem: files, Args: []string{"cc", "main.c"}, Target: "linux/amd64"})
	if err != nil || !result.Ok {
		t.Fatalf("compile: %+v %v", result, err)
	}
	executable := filepath.Join(t.TempDir(), "headers")
	if err := os.WriteFile(executable, result.Binary, 0700); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable)
	command.Env = []string{"LC_ALL=C"}
	if output, err := command.CombinedOutput(); err != nil || string(output) != "PASS\n" {
		t.Fatalf("run: %v %q", err, output)
	}
}
