package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/std/vm"
)

func TestCompilerIntrinsicAliasVM(t *testing.T) {
	source, err := os.ReadFile("tests/intrinsic_declaration_identity.go")
	if err != nil {
		t.Fatal(err)
	}
	resetRuntime()
	image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/vm32", RenvoCompileOptions{
		ArenaSize: 8 * 1024 * 1024, StripSymbols: true,
	})
	if !ok {
		t.Fatal("compile VM intrinsic declaration")
	}
	result := vm.RunConfig(image, vm.Config{Limits: vm.Limits{Steps: 500 * 1000 * 1000, Memory: 16 * 1024 * 1024}})
	if result.Trap != vm.TrapNone || result.ExitCode != 0 || string(result.Output)+string(result.Stderr) != "PASS\n" {
		t.Fatalf("exit %d, trap %d, stdout %q, stderr %q", result.ExitCode, result.Trap, result.Output, result.Stderr)
	}
}

func TestCompilerIntrinsicSyscallAliasNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires a native Linux AMD64 runner")
	}
	for _, export := range []string{"", "//export processID\n"} {
		t.Run(export, func(t *testing.T) {
			resetRuntime()
			source := []byte("package main\n" +
				"func renvo_runtime_Syscall_other()int{return 17}\n" +
				"//renvo:intrinsic renvo_runtime_Syscall\n" + export +
				"func renvo_runtime_Syscall_(number,fd int,msg string,n int)int{return -1}\n" +
				"func appMain()int{if renvo_runtime_Syscall_(39,0,\"\",0)<=0||renvo_runtime_Syscall_other()!=17{return 1};print(\"PASS\\n\");return 0}\n")
			image, ok := RenvoCompileSourceToBytes(source, "linux/amd64")
			if !ok {
				t.Fatal("compile syscall alias")
			}
			path := filepath.Join(t.TempDir(), "program")
			if err := os.WriteFile(path, image, 0755); err != nil {
				t.Fatal(err)
			}
			output, err := exec.Command(path).CombinedOutput()
			if err != nil || string(output) != "PASS\n" {
				t.Fatalf("run: %v, output %q", err, output)
			}
		})
	}
}
