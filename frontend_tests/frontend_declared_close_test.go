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

func TestFrontendDeclaredClose(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("Linux/amd64")
	}
	source, e := os.ReadFile(filepath.Join(repoRoot(t), "backend", "tests", "close_declared_function.go"))
	if e != nil {
		t.Fatal(e)
	}
	fs := linkSourceFS{fstest.MapFS{"main.go": {Data: source}}}
	r, e := driver.CompileCommand(&driver.CommandRequest{Filesystem: fs, Args: []string{"main.go", "-o", "app"}, Target: "linux/amd64"})
	if e != nil || !r.Ok {
		t.Fatalf("%+v %v", r, e)
	}
	executable := filepath.Join(t.TempDir(), "close")
	if e := os.WriteFile(executable, r.Binary, 0700); e != nil {
		t.Fatal(e)
	}
	if output, e := exec.Command(executable).CombinedOutput(); e != nil || string(output) != "PASS\n" {
		t.Fatalf("%v %q", e, output)
	}
}
func TestFrontendCFcntlObject(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("Linux/amd64")
	}
	source, e := os.ReadFile(filepath.Join(repoRoot(t), "libc", "tests", "fcntl_object.c"))
	if e != nil {
		t.Fatal(e)
	}
	fs := linkSourceFS{fstest.MapFS{"main.c": {Data: source}}}
	object, e := driver.CompileCommand(&driver.CommandRequest{Filesystem: fs, Args: []string{"cc", "-c", "main.c"}, Target: "linux/amd64"})
	if e != nil || !object.Ok {
		t.Fatalf("%+v %v", object, e)
	}
	fs.MapFS["main.o"] = &fstest.MapFile{Data: object.Binary}
	r, e := driver.CompileCommand(&driver.CommandRequest{Filesystem: fs, Args: []string{"cc", "main.o"}, Target: "linux/amd64"})
	if e != nil || !r.Ok {
		t.Fatalf("%+v %v", r, e)
	}
	executable := filepath.Join(t.TempDir(), "fcntl")
	if e := os.WriteFile(executable, r.Binary, 0700); e != nil {
		t.Fatal(e)
	}
	if output, e := exec.Command(executable).CombinedOutput(); e != nil {
		t.Fatalf("%v %q", e, output)
	}
}
