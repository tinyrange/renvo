package rtgx

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCompileSourceBuildsRunnableExecutable(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 executable smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, ok := findBackendRootUpward(".")
	if !ok {
		t.Skip("backend root not found")
	}
	out := filepath.Join(t.TempDir(), "app")
	src := []byte(`//go:build rtg

package main

func appMain() int {
	print("PASS\n")
	return 0
}
`)
	if err := CompileSource(src, Options{Target: "linux/amd64", Output: out, BackendRoot: root}); err != nil {
		t.Fatalf("CompileSource failed: %v", err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("Stat output failed: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Fatalf("output is not executable: %v", info.Mode())
	}
	cmd := exec.Command(out)
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled app failed: %v\n%s", err, string(data))
	}
	if string(data) != "PASS\n" {
		t.Fatalf("compiled app output = %q", string(data))
	}
}
