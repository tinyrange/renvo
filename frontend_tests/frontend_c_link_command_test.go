package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"

	"renvo.dev/driver"
)

type linkSourceFS struct{ fstest.MapFS }

func (f linkSourceFS) ReadFile(name string) ([]byte, bool) {
	b, e := f.MapFS.ReadFile(name)
	return b, e == nil
}
func (f linkSourceFS) PathExists(name string) bool { _, e := f.MapFS.Stat(name); return e == nil }
func (f linkSourceFS) ReadDir(name string) ([]driver.DirEntry, bool) {
	entries, e := f.MapFS.ReadDir(name)
	if e != nil {
		return nil, false
	}
	var out []driver.DirEntry
	for _, v := range entries {
		out = append(out, driver.DirEntry{Name: v.Name(), IsDir: v.IsDir()})
	}
	return out, true
}

// C object main must remain an ordinary exported function, not Renvo's hosted
// appMain entry with its different internal parameter-binding convention.
func TestFrontendCObjectMainLinkCommand(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("native Linux/amd64 acceptance")
	}
	source, err := os.ReadFile(filepath.Join(repoRoot(t), "backend", "tests", "c_object_main_arguments.c"))
	if err != nil {
		t.Fatal(err)
	}
	files := linkSourceFS{fstest.MapFS{
		"main.c":   {Data: source},
		"answer.c": {Data: []byte(`int answer(int n){return n+40;}`)},
	}}
	for _, name := range []string{"main", "answer"} {
		r, e := driver.CompileCommand(&driver.CommandRequest{Filesystem: files, Args: []string{"cc", "-c", name + ".c", "-o", name + ".o"}, Target: "linux/amd64", ArenaSize: 1 << 20})
		if e != nil || !r.Ok {
			t.Fatalf("%s: %+v %v", name, r, e)
		}
		files.MapFS[name+".o"] = &fstest.MapFile{Data: r.Binary}
	}
	r, e := driver.LinkCommand(&driver.CommandRequest{Filesystem: files, Args: []string{"main.o", "answer.o"}, Target: "linux/amd64"})
	if e != nil || !r.Ok {
		t.Fatalf("link: %+v %v", r, e)
	}
	// Only the final executable is materialized for native acceptance. Sources
	// and object/link intermediates remain in memory throughout the pipeline.
	executable := filepath.Join(t.TempDir(), "linked")
	if e := os.WriteFile(executable, r.Binary, 0700); e != nil {
		t.Fatal(e)
	}
	if output, e := exec.Command(executable, "x").CombinedOutput(); e != nil || len(output) != 0 {
		t.Fatalf("native: %v %q", e, output)
	}
}
