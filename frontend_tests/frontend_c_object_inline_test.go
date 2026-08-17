package frontend_tests

import (
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendCObjectGNUExternInlineSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectGNUExternInlineSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectGNUExternInlineSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectGNUExternInlineSystemLink(t, selfHostedFrontendCompiler(t, root))
}

func runCObjectGNUExternInlineSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}

	dir := t.TempDir()
	firstSource := filepath.Join(dir, "first.c")
	secondSource := filepath.Join(dir, "second.c")
	harness := filepath.Join(dir, "harness.c")
	firstObject := filepath.Join(dir, "first.o")
	secondObject := filepath.Join(dir, "second.o")
	executable := filepath.Join(dir, "inline-test")
	definition := "extern inline __attribute__((__gnu_inline__)) int header_add(int value) { return value + 1; }\n"
	if err := os.WriteFile(firstSource, []byte(definition+"int first_add(int value) { return header_add(value); }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondSource, []byte(definition+"int second_add(int value) { return header_add(value); }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte("int first_add(int); int second_add(int); int main(void) { return first_add(3) == 4 && second_add(8) == 9 ? 0 : 1; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{firstSource, firstObject}, {secondSource, secondObject}} {
		command := frontendCommand(frontend, "cc", "-c", filepath.Base(pair[0]), "-o", pair[1])
		command.Dir = dir
		command.Env = frontendCommandEnv(frontend.env, dir)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("compile GNU extern-inline object with Renvo: %v\n%s", err, output)
		}
		file, err := elf.Open(pair[1])
		if err != nil {
			t.Fatal(err)
		}
		symbols, err := file.Symbols()
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		for _, symbol := range symbols {
			if symbol.Name == "header_add" && elf.ST_BIND(symbol.Info) == elf.STB_GLOBAL {
				t.Fatal("GNU extern-inline helper has global linkage")
			}
		}
	}
	link := exec.Command(linkerDriver, harness, firstObject, secondObject, "-o", executable)
	if output, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link GNU extern-inline objects: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run GNU extern-inline objects: %v, output %q", err, output)
	}
}
