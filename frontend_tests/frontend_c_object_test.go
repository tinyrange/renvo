package frontend_tests

import (
	"debug/elf"
	"encoding/binary"
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

func TestFrontendCObjectWithoutMainSystemLink(t *testing.T) {
	root := repoRoot(t)
	frontend := frontendCompiler(t, root)
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "bridge.c")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "bridge.o")
	executable := filepath.Join(dir, "harness")
	if err := os.WriteFile(source,
		[]byte("int renvo_weighted(int left, int right) { return left * 100 + right; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness,
		[]byte("extern int renvo_weighted(int, int);\nint main(void) { return renvo_weighted(50, 8) == 5008 ? 0 : 1; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C leaf object with Renvo: %v\n%s", err, combined)
	}
	link := exec.Command(linkerDriver, harness, object, "-o", executable)
	if combined, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link Renvo C leaf object: %v\n%s", err, combined)
	}
	if combined, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked C leaf object: %v, output %q", err, combined)
	}
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
	if err := os.WriteFile(source, []byte("#include <stdio.h>\nint main(void) { puts(\"Hello, world!\"); return 0; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
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
	putsSymbol := -1
	for i, symbol := range symbols {
		if symbol.Name == "_printk" {
			t.Fatal("ordinary C object imports the kernel module runtime")
		}
		if symbol.Name == "main" && elf.ST_BIND(symbol.Info) == elf.STB_GLOBAL && symbol.Section != elf.SHN_UNDEF {
			foundMain = true
		}
		if symbol.Name == "puts" && elf.ST_BIND(symbol.Info) == elf.STB_GLOBAL && symbol.Section == elf.SHN_UNDEF {
			// debug/elf omits the mandatory null symbol from Symbols().
			putsSymbol = i + 1
		}
	}
	if !foundMain {
		t.Fatal("C object does not export a defined global main")
	}
	if putsSymbol < 0 {
		t.Fatal("C object does not contain an undefined global puts")
	}
	relocations := file.Section(".rela.text")
	if relocations == nil {
		t.Fatal("C object has no .rela.text section")
	}
	relocationData, err := relocations.Data()
	if err != nil {
		t.Fatal(err)
	}
	foundPutsRelocation := false
	for at := 0; at+24 <= len(relocationData); at += 24 {
		info := binary.LittleEndian.Uint64(relocationData[at+8 : at+16])
		if int(info>>32) == putsSymbol && elf.R_X86_64(uint32(info)) == elf.R_X86_64_PLT32 {
			foundPutsRelocation = true
			break
		}
	}
	if !foundPutsRelocation {
		t.Fatal("C object has no R_X86_64_PLT32 relocation for puts")
	}

	link := exec.Command(linkerDriver, object, "-o", executable)
	if combined, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link Renvo C object: %v\n%s", err, combined)
	}
	if combined, err := exec.Command(executable).CombinedOutput(); err != nil || string(combined) != "Hello, world!\n" {
		t.Fatalf("run linked C object: %v, output %q", err, combined)
	}
}
