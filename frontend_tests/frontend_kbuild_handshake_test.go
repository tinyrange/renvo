package frontend_tests

import (
	"debug/elf"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFrontendKbuildHandshake(t *testing.T) {
	root := repoRoot(t)
	runKbuildHandshake(t, frontendCompiler(t, root))
}

func TestFrontendStage3KbuildHandshake(t *testing.T) {
	root := repoRoot(t)
	runKbuildHandshake(t, selfHostedFrontendCompiler(t, root))
}

func runKbuildHandshake(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the pinned Kbuild handshake target is Linux/amd64")
	}

	version := frontendCommand(frontend, "cc", "--version")
	if output, err := version.CombinedOutput(); err != nil || string(output) != "renvo cc (GCC-compatible) 5.1.0\n" {
		t.Fatalf("compiler version query: %v, output %q", err, output)
	}

	probe := frontendCommand(frontend, "cc", "-E", "-P", "-x", "c", "-")
	probe.Stdin = strings.NewReader("#if defined(__GNUC__)\nGCC __GNUC__ __GNUC_MINOR__ __GNUC_PATCHLEVEL__\n#else\nunknown\n#endif\n")
	if output, err := probe.CombinedOutput(); err != nil || string(output) != "GCC 5 1 0\n" {
		t.Fatalf("compiler identity preprocess: %v, output %q", err, output)
	}

	dir := t.TempDir()
	include := filepath.Join(dir, "include")
	if err := os.MkdirAll(include, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "empty.c"), []byte("/* Kbuild target compiler handshake. */\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(include, "compiler-version.h"), []byte("#define RENVO_KERNEL_COMPILER 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	common := "-Wp,-MMD,.empty.o.d -nostdinc -Iinclude -include include/compiler-version.h " +
		"-D__KERNEL__ -std=gnu11 -fshort-wchar -funsigned-char -fno-common -fno-PIE " +
		"-fno-strict-aliasing -mno-sse -m64 -mno-red-zone -mcmodel=kernel -Os " +
		"-fno-stack-protector -falign-functions=16 -Wall -Werror=implicit-function-declaration\n"
	if err := os.WriteFile(filepath.Join(dir, "common.rsp"), []byte(common), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "compile.rsp"), []byte("@common.rsp -c empty.c -o empty.o\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	compile := frontendCommand(frontend, "cc", "@compile.rsp")
	compile.Dir = dir
	compile.Env = frontendCommandEnv(frontend.env, dir)
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("Kbuild-shaped empty object: %v\n%s", err, output)
	}
	object, err := elf.Open(filepath.Join(dir, "empty.o"))
	if err != nil {
		t.Fatal(err)
	}
	if object.Type != elf.ET_REL || object.Machine != elf.EM_X86_64 {
		object.Close()
		t.Fatalf("empty object type/machine = %v/%v", object.Type, object.Machine)
	}
	object.Close()
	dependency, err := os.ReadFile(filepath.Join(dir, ".empty.o.d"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dependency), "empty.c") || !strings.Contains(string(dependency), "include/compiler-version.h") {
		t.Fatalf("dependency rule = %q", dependency)
	}

	unknown := frontendCommand(frontend, "cc", "-frenvo-pretend-support", "-c", "empty.c", "-o", "unknown.o")
	unknown.Dir = dir
	unknown.Env = frontendCommandEnv(frontend.env, dir)
	if output, err := unknown.CombinedOutput(); err == nil || !strings.Contains(string(output), "unknown option -frenvo-pretend-support") {
		t.Fatalf("unknown option probe unexpectedly succeeded: %v, output %q", err, output)
	}
}
