package frontend_tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestUnsafePointerUintptrStructLayout386(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" && runtime.GOARCH != "386" {
		t.Skipf("linux/386 regression requires an x86 Linux host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root := repoRoot(t)
	frontend := frontendCompiler(t, root)
	if frontend.compiler == "" {
		t.Skip("frontend compiler is unavailable")
	}
	frontend.target = "linux/386"
	runFrontendCorpusCase(t, frontend, filepath.Join(root, "frontend_tests", "regressions", "unsafe_pointer_uintptr_struct_layout"))
}
