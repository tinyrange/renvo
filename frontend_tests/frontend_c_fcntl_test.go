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

func TestFrontendCFcntl(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("Linux/amd64")
	}
	source, e := os.ReadFile(filepath.Join(repoRoot(t), "libc", "tests", "fcntl.c"))
	if e != nil {
		t.Fatal(e)
	}
	fs := linkSourceFS{fstest.MapFS{"main.c": {Data: source}}}
	r, e := driver.CompileCommand(&driver.CommandRequest{Filesystem: fs, Args: []string{"cc", "main.c"}, Target: "linux/amd64"})
	if e != nil || !r.Ok {
		t.Fatalf("%+v %v", r, e)
	}
	dir := t.TempDir()
	executable := filepath.Join(dir, "fcntl")
	if e := os.WriteFile(executable, r.Binary, 0700); e != nil {
		t.Fatal(e)
	}
	c := exec.Command(executable)
	c.Dir = dir
	if output, e := c.CombinedOutput(); e != nil || string(output) != "PASS\n" {
		t.Fatalf("%v %q", e, output)
	}
}
