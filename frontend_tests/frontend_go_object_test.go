package frontend_tests

import (
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendGoObjectSystemLink(t *testing.T) {
	root := repoRoot(t)
	runGoObjectSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3GoObjectSystemLink(t *testing.T) {
	root := repoRoot(t)
	runGoObjectSystemLink(t, selfHostedFrontendCompiler(t, root))
}

func runGoObjectSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first Go object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "bridge.go")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "bridge.o")
	executable := filepath.Join(dir, "harness")
	goSource := "package bridge\n\n//export renvo_weighted\nfunc Weighted(left int64, right int64) int64 { return left*100 + right }\n"
	cSource := "#include <stdint.h>\nextern int64_t renvo_weighted(int64_t, int64_t);\nint main(void) { return renvo_weighted(50, 8) == 5008 ? 0 : 1; }\n"
	if err := os.WriteFile(source, []byte(goSource), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(cSource), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend,
		"-mode=object", "-o", object, filepath.Base(source))
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile Go object with Renvo: %v\n%s", err, combined)
	}

	file, err := elf.Open(object)
	if err != nil {
		t.Fatalf("open Renvo Go object: %v", err)
	}
	defer file.Close()
	if file.Type != elf.ET_REL || file.Machine != elf.EM_X86_64 {
		t.Fatalf("ELF type/machine = %v/%v", file.Type, file.Machine)
	}
	foundExport := false
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Name == "renvo_weighted" && elf.ST_BIND(symbol.Info) == elf.STB_GLOBAL &&
			symbol.Section != elf.SHN_UNDEF {
			foundExport = true
		}
	}
	if !foundExport {
		t.Fatal("Go object does not export renvo_weighted")
	}

	link := exec.Command(linkerDriver, harness, object, "-o", executable)
	if combined, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link Renvo Go object: %v\n%s", err, combined)
	}
	if combined, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked Go object: %v, output %q", err, combined)
	}
}
