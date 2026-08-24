package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreprocessingCommandRewritesOnlyCompileSuffix(t *testing.T) {
	command := "cc -nostdinc -Iinclude -DNAME='quoted value' -c -o build/main.o source/main.c"
	source, rewritten, ok := preprocessingCommand(command, "/tmp/work dir/unit.i")
	if !ok {
		t.Fatal("preprocessing command was rejected")
	}
	if source != "source/main.c" {
		t.Fatalf("source = %q", source)
	}
	if !strings.HasPrefix(rewritten, "cc -nostdinc -Iinclude -DNAME='quoted value'") {
		t.Fatalf("flags changed: %q", rewritten)
	}
	if !strings.HasSuffix(rewritten, " -E -P source/main.c -o '/tmp/work dir/unit.i'") {
		t.Fatalf("compile suffix = %q", rewritten)
	}
}

func TestPreprocessingCommandRejectsUnexpectedSuffix(t *testing.T) {
	if _, _, ok := preprocessingCommand("cc -c source/main.c -o build/main.o", "unit.i"); ok {
		t.Fatal("accepted command whose Kbuild compile suffix is not supported")
	}
}

func TestObjectArgumentsPreserveSemanticCodeGenerationFlags(t *testing.T) {
	arguments := objectArguments("cc -m16 -fshort-wchar -mcmodel=kernel -c -o main.o main.c", "out.o", "unit.c")
	joined := strings.Join(arguments, " ")
	for _, want := range []string{"-fshort-wchar", "-mcmodel=kernel", "-m16"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("object arguments %q are missing %q", joined, want)
		}
	}
	plain := strings.Join(objectArguments("cc -m64 -c -o main.o main.c", "out.o", "unit.c"), " ")
	if strings.Contains(plain, "-mcmodel=kernel") || strings.Contains(plain, "-fshort-wchar") {
		t.Fatalf("plain object arguments retained semantic flags: %q", plain)
	}
}

func TestTargetObjectPath(t *testing.T) {
	path, ok := targetObjectPath("cc -m64 -c -o init/main.o init/main.c")
	if !ok || path != "init/main.o" {
		t.Fatalf("targetObjectPath = %q, %v", path, ok)
	}
	for _, command := range []string{
		"cc -m64 -c init/main.c -o init/main.o",
		"cc -m64 -c -o /tmp/main.o init/main.c",
		"cc -m64 -c -o ../main.o init/main.c",
	} {
		if path, ok := targetObjectPath(command); ok {
			t.Fatalf("targetObjectPath(%q) = %q, true", command, path)
		}
	}
}

func TestAuditDirectObjectsRequiresSelectedCompilerAndELF(t *testing.T) {
	kernel := t.TempDir()
	compiler := filepath.Join(t.TempDir(), "renvo-cc")
	object := filepath.Join(kernel, "init", "main.o")
	if err := os.MkdirAll(filepath.Dir(object), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(object, []byte("\x7fELFobject"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := compiler + " -nostdinc -D__KERNEL__ -c -o init/main.o init/main.c"
	if err := auditDirectObjects(kernel, compiler, []string{command}); err != nil {
		t.Fatal(err)
	}
	if err := auditDirectObjects(kernel, "/usr/bin/cc", []string{command}); err == nil || !strings.Contains(err.Error(), "command used") {
		t.Fatalf("compiler mismatch error = %v", err)
	}
	if err := os.WriteFile(object, []byte("not ELF"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := auditDirectObjects(kernel, compiler, []string{command}); err == nil || !strings.Contains(err.Error(), "is not ELF") {
		t.Fatalf("non-ELF error = %v", err)
	}
}

func TestM16TargetCommandCountRecognizesOnlyExactOption(t *testing.T) {
	commands := []string{
		"renvo-cc -m64 -c -o init/main.o init/main.c",
		"renvo-cc -m16 -c -o arch/x86/boot/main.o arch/x86/boot/main.c",
		"renvo-cc -DVALUE=-m16 -c -o lib/value.o lib/value.c",
	}
	if got := m16TargetCommandCount(commands); got != 1 {
		t.Fatalf("m16TargetCommandCount = %d, want 1", got)
	}
}

func TestTargetCompileCommandDropsPostCompileObjtool(t *testing.T) {
	command := "gcc -nostdinc -D__KERNEL__ -c -o init/main.o init/main.c   ; ./tools/objtool/objtool --uaccess init/main.o"
	want := "gcc -nostdinc -D__KERNEL__ -c -o init/main.o init/main.c"
	if got := targetCompileCommand(command); got != want {
		t.Fatalf("targetCompileCommand = %q, want %q", got, want)
	}
}

func TestFilterTargetCommandsMatchesSourcePath(t *testing.T) {
	commands := []string{
		"gcc -D__KERNEL__ -c -o init/main.o init/main.c",
		"gcc -D__KERNEL__ -c -o mm/show_mem.o mm/show_mem.c",
	}
	selected := filterTargetCommands(commands, "show_mem")
	if len(selected) != 1 || selected[0] != commands[1] {
		t.Fatalf("filtered commands = %q", selected)
	}
}

func TestInstallVmlinuxTargetObjectsReplacesOnlyRecordedRegularFiles(t *testing.T) {
	kernel := t.TempDir()
	workspace := t.TempDir()
	destination := filepath.Join(kernel, "init", "main.o")
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("reference"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "object-007.o"), []byte("renvo"), 0o600); err != nil {
		t.Fatal(err)
	}
	jobs := []syntaxJob{{index: 7, source: "init/main.c", command: "cc -c -o init/main.o init/main.c"}}
	installed, err := installVmlinuxTargetObjects(kernel, workspace, jobs)
	if err != nil {
		t.Fatal(err)
	}
	if installed != 1 {
		t.Fatalf("installed = %d, want 1", installed)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "renvo" {
		t.Fatalf("installed object = %q", data)
	}
	info, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("installed mode = %v", got)
	}

	bad := filepath.Join(kernel, "init", "bad.o")
	if err := os.Mkdir(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "object-008.o"), []byte("bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = installVmlinuxTargetObjects(kernel, workspace, []syntaxJob{{index: 8, source: "init/bad.c", command: "cc -c -o init/bad.o init/bad.c"}})
	if err == nil || !strings.Contains(err.Error(), "non-regular") {
		t.Fatalf("non-regular destination error = %v", err)
	}
}

func TestInstallVmlinuxTargetObjectsLeavesIndependentX86ImagesUntouched(t *testing.T) {
	kernel := t.TempDir()
	workspace := t.TempDir()
	jobs := []syntaxJob{
		{index: 2, source: "arch/x86/boot/main.c", command: "cc -c -o arch/x86/boot/main.o arch/x86/boot/main.c"},
		{index: 3, source: "arch/x86/entry/vdso/vgetcpu.c", command: "cc -c -o arch/x86/entry/vdso/vgetcpu.o arch/x86/entry/vdso/vgetcpu.c"},
	}
	for _, job := range jobs {
		relative, ok := targetObjectPath(job.command)
		if !ok {
			t.Fatalf("invalid test command %q", job.command)
		}
		destination := filepath.Join(kernel, relative)
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, []byte("reference"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workspace, fmt.Sprintf("object-%03d.o", job.index)), []byte("renvo"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	installed, err := installVmlinuxTargetObjects(kernel, workspace, jobs)
	if err != nil {
		t.Fatal(err)
	}
	if installed != 0 {
		t.Fatalf("installed = %d, want 0", installed)
	}
	for _, job := range jobs {
		relative, _ := targetObjectPath(job.command)
		data, err := os.ReadFile(filepath.Join(kernel, relative))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "reference" {
			t.Fatalf("independent image object %s = %q", relative, data)
		}
	}
}
