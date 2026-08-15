package frontend_tests

import (
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendCObjectSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectSystemLink(t, selfHostedFrontendCompiler(t, root))
}

func runCObjectSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "hello.c")
	object := filepath.Join(dir, "hello.o")
	executable := filepath.Join(dir, "hello")
	if err := os.WriteFile(source, []byte("int main(void) { return 0; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", source, "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C object with Renvo: %v\n%s", err, combined)
	}

	file, err := elf.Open(object)
	if err != nil {
		t.Fatalf("open Renvo C object: %v", err)
	}
	defer file.Close()
	if file.Type != elf.ET_REL || file.Machine != elf.EM_X86_64 {
		t.Fatalf("ELF type/machine = %v/%v", file.Type, file.Machine)
	}
	if file.Section(".modinfo") != nil || file.Section(".gnu.linkonce.this_module") != nil {
		t.Fatal("ordinary C object contains Linux module metadata")
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	foundMain := false
	for _, symbol := range symbols {
		if symbol.Name == "_printk" {
			t.Fatal("ordinary C object imports the kernel module runtime")
		}
		if symbol.Name == "main" && elf.ST_BIND(symbol.Info) == elf.STB_GLOBAL && symbol.Section != elf.SHN_UNDEF {
			foundMain = true
		}
	}
	if !foundMain {
		t.Fatal("C object does not export a defined global main")
	}

	link := exec.Command(linkerDriver, object, "-o", executable)
	if combined, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link Renvo C object: %v\n%s", err, combined)
	}
	if combined, err := exec.Command(executable).CombinedOutput(); err != nil || len(combined) != 0 {
		t.Fatalf("run linked C object: %v, output %q", err, combined)
	}
}
