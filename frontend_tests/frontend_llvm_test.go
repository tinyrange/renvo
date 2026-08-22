package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLLVMBackendSelfHosting(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("LLVM backend execution requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("LLVM backend execution requires clang")
	}

	root := repoRoot(t)
	work := t.TempDir()
	buildGoTool(t, root, filepath.Join(work, "renvo-backend"), nil, "./backend")
	stage0 := filepath.Join(work, "renvo")
	buildGoTool(t, root, stage0, nil, "./cmd/renvo")

	stage1IR := filepath.Join(work, "stage1.ll")
	runLLVMCommand(t, root, stage0,
		"-backend", filepath.Join(root, "backends", "llvm_amd64.rtg"),
		"-t", "llvm/linux-amd64", "-tags", "renvo_prepared",
		"-arena-size", "1073741824", "-s", "-o", stage1IR, "./cmd/renvo")
	stage1 := filepath.Join(work, "stage1")
	runLLVMCommand(t, root, clang, "-O0", "-Wno-override-module", "-o", stage1, stage1IR)

	stage2IR := filepath.Join(work, "stage2.ll")
	runLLVMCommand(t, root, stage1,
		"-t", "llvm/linux-amd64", "-tags", "renvo_prepared",
		"-arena-size", "1073741824", "-s", "-o", stage2IR, "./cmd/renvo")
	stage2 := filepath.Join(work, "stage2")
	runLLVMCommand(t, root, clang, "-O0", "-Wno-override-module", "-o", stage2, stage2IR)

	helloIR := filepath.Join(work, "hello.ll")
	runLLVMCommand(t, root, stage2,
		"-t", "llvm/linux-amd64", "-s", "-o", helloIR, "./tools/wasm/testdata")
	hello := filepath.Join(work, "hello")
	runLLVMCommand(t, root, clang, "-O2", "-Wno-override-module", "-o", hello, helloIR)
	cmd := exec.Command(hello)
	if output, err := cmd.CombinedOutput(); err != nil || string(output) != "PASS\n" {
		t.Fatalf("LLVM-compiled program failed: err=%v output=%q", err, output)
	}
}

func runLLVMCommand(t *testing.T, root string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PWD="+root, "RENVO_STDROOT="+filepath.Join(root, "std"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s failed: %v\n%s", filepath.Base(name), err, output)
	}
}
