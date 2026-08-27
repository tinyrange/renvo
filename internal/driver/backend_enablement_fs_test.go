package driver

import (
	"testing"

	"renvo.dev/internal/load"
	"renvo.dev/internal/rbe"
)

func TestBackendEnablementFSAddsAndExtendsStandardPackages(t *testing.T) {
	base := memorySourceFS{files: []load.SourceFile{
		{Path: "/std/os/base.go", Src: []byte("package os\n")},
	}}
	fs := backendEnablementFS{base: base, stdRoot: "/std", files: []rbe.File{
		{Path: "os/v7.go", Source: []byte("package os\nconst V7 = true\n")},
		{Path: "syscall/syscall.go", Source: []byte("package syscall\n")},
	}}
	root, ok := fs.ReadDir("/std")
	if !ok || !hasDirEntry(root, "os", true) || !hasDirEntry(root, "syscall", true) {
		t.Fatalf("root entries = %#v, ok=%v", root, ok)
	}
	osFiles, ok := fs.ReadDir("/std/os")
	if !ok || !hasDirEntry(osFiles, "base.go", false) || !hasDirEntry(osFiles, "v7.go", false) {
		t.Fatalf("os entries = %#v, ok=%v", osFiles, ok)
	}
	if source, ok := fs.ReadFile("/std/syscall/syscall.go"); !ok || string(source) != "package syscall\n" {
		t.Fatalf("ReadFile = %q, %v", source, ok)
	}
	if !fs.PathExists("/std/syscall") || !fs.PathExists("/std/syscall/syscall.go") {
		t.Fatal("overlay paths do not exist")
	}
}

func hasDirEntry(entries []DirEntry, name string, directory bool) bool {
	for _, entry := range entries {
		if entry.Name == name && entry.IsDir == directory {
			return true
		}
	}
	return false
}
