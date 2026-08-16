package frontend_tests

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

func TestFrontendKernelObjectContract(t *testing.T) {
	root := repoRoot(t)
	runKernelObjectContract(t, frontendCompiler(t, root))
}

func TestFrontendStage3KernelObjectContract(t *testing.T) {
	root := repoRoot(t)
	runKernelObjectContract(t, selfHostedFrontendCompiler(t, root))
}

func TestFrontendCustomRTGKernelObjectContract(t *testing.T) {
	root := repoRoot(t)
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first kernel object target is Linux/amd64")
	}
	frontend := integratedFrontendCompiler(t, root)
	runKernelObjectContract(t, frontend,
		"-backend", filepath.Join(root, "backends", "linux_amd64_object.rtg"),
		"-t", "linux-object/amd64")
}

func runKernelObjectContract(t *testing.T, frontend frontendConfig, backendArgs ...string) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first kernel object target is Linux/amd64")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "leaf.c")
	object := filepath.Join(dir, "leaf.o")
	dependency := filepath.Join(dir, "leaf.d")
	const program = `
extern int external_call(int);
static int local_bss __attribute__((section(".bss.renvo"), aligned(32)));
int public_data __attribute__((section(".data.renvo"), aligned(16))) = 7;
const int read_only __attribute__((section(".rodata.renvo"))) = 9;
__attribute__((section(".text.renvo"), aligned(32), visibility("hidden")))
int leaf(int value) {
	local_bss += value;
	return local_bss + public_data + read_only + external_call(value);
}
int weak_leaf(int value) __attribute__((weak, alias("leaf")));
int linkonce_leaf(int value) __attribute__((weak, section(".gnu.linkonce.t.linkonce_leaf")));
int linkonce_leaf(int value) { return leaf(value) + 1; }
int *absolute64 = &public_data;
unsigned absolute32 = (unsigned long)&public_data;
int absolute32s = (long)&public_data;
`
	if err := os.WriteFile(source, []byte(program), 0o644); err != nil {
		t.Fatal(err)
	}
	compile := func() []byte {
		compileArgs := []string{"cc"}
		compileArgs = append(compileArgs, backendArgs...)
		compileArgs = append(compileArgs, "-MMD", "-ffunction-sections", "-fdata-sections", "-c", "leaf.c", "-o", "leaf.o")
		command := frontendCommand(frontend, compileArgs...)
		command.Dir = dir
		command.Env = frontendCommandEnv(frontend.env, dir)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("compile kernel-shaped leaf object: %v\n%s", err, output)
		}
		data, err := os.ReadFile(object)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	first := compile()
	depFirst, err := os.ReadFile(dependency)
	if err != nil {
		t.Fatal(err)
	}
	if string(depFirst) != "leaf.o: leaf.c\n" {
		t.Fatalf("dependency fixture = %q", depFirst)
	}
	second := compile()
	depSecond, err := os.ReadFile(dependency)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) || !bytes.Equal(depFirst, depSecond) {
		t.Fatal("object or dependency output is nondeterministic")
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.c"), []byte("int broken( {\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	broken := frontendCommand(frontend, append(append([]string{"cc"}, backendArgs...), "-MMD", "-c", "broken.c", "-o", "broken.o")...)
	broken.Dir = dir
	broken.Env = frontendCommandEnv(frontend.env, dir)
	if output, err := broken.CombinedOutput(); err == nil {
		t.Fatalf("invalid C object unexpectedly compiled:\n%s", output)
	}
	if _, err := os.Stat(filepath.Join(dir, "broken.d")); !os.IsNotExist(err) {
		t.Fatalf("failed compile published a dependency file: %v", err)
	}

	file, err := elf.Open(object)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if file.Type != elf.ET_REL || file.Machine != elf.EM_X86_64 {
		t.Fatalf("ELF type/machine = %v/%v", file.Type, file.Machine)
	}
	sections := map[string]*elf.Section{}
	for _, section := range file.Sections {
		sections[section.Name] = section
	}
	for name, alignment := range map[string]uint64{
		".text.renvo": 32, ".gnu.linkonce.t.linkonce_leaf": 16,
		".bss.renvo": 32, ".data.renvo": 16, ".rodata.renvo": 4,
	} {
		section := sections[name]
		if section == nil || section.Addralign != alignment {
			t.Fatalf("section %s = %#v, want alignment %d", name, section, alignment)
		}
	}
	if sections[".bss.renvo"].Type != elf.SHT_NOBITS || sections[".rodata.renvo"].Flags&elf.SHF_WRITE != 0 ||
		sections[".gnu.linkonce.t.linkonce_leaf"].Flags&elf.SHF_GROUP == 0 || sections[".group"] == nil {
		t.Fatal("section type/flag/group fixture mismatch")
	}

	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]elf.Symbol{}
	for _, symbol := range symbols {
		byName[symbol.Name] = symbol
	}
	leaf, weak := byName["leaf"], byName["weak_leaf"]
	if elf.ST_BIND(leaf.Info) != elf.STB_GLOBAL || elf.ST_VISIBILITY(leaf.Other) != elf.STV_HIDDEN || leaf.Size == 0 {
		t.Fatalf("leaf symbol = %#v", leaf)
	}
	if elf.ST_BIND(weak.Info) != elf.STB_WEAK || weak.Section != leaf.Section || weak.Value != leaf.Value {
		t.Fatalf("weak alias = %#v, leaf %#v", weak, leaf)
	}
	if symbol := byName["local_bss"]; elf.ST_BIND(symbol.Info) != elf.STB_LOCAL || symbol.Size != 4 {
		t.Fatalf("local BSS symbol = %#v", symbol)
	}
	for _, name := range []string{"public_data", "read_only", "absolute64", "absolute32", "absolute32s"} {
		if symbol := byName[name]; elf.ST_BIND(symbol.Info) != elf.STB_GLOBAL || symbol.Section == elf.SHN_UNDEF {
			t.Fatalf("data symbol %s = %#v", name, symbol)
		}
	}
	if symbol := byName["external_call"]; elf.ST_BIND(symbol.Info) != elf.STB_GLOBAL || symbol.Section != elf.SHN_UNDEF {
		t.Fatalf("external symbol = %#v", symbol)
	}

	var relocationKinds []int
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			info := binary.LittleEndian.Uint64(data[at+8 : at+16])
			relocationKinds = append(relocationKinds, int(uint32(info)))
		}
	}
	sort.Ints(relocationKinds)
	for _, wanted := range []int{int(elf.R_X86_64_64), int(elf.R_X86_64_PC32), int(elf.R_X86_64_PLT32), int(elf.R_X86_64_32), int(elf.R_X86_64_32S)} {
		if !containsRelocationKind(relocationKinds, wanted) {
			t.Fatalf("relocations %v do not contain %v", relocationKinds, elf.R_X86_64(wanted))
		}
	}

	archiver, err := exec.LookPath("ar")
	if err != nil {
		t.Skip("system archiver is unavailable")
	}
	archive := filepath.Join(dir, "built-in.a")
	if output, err := exec.Command(archiver, "rcsD", archive, object).CombinedOutput(); err != nil {
		t.Fatalf("archive Renvo leaf object: %v\n%s", err, output)
	}
	if output, err := exec.Command(archiver, "t", archive).CombinedOutput(); err != nil || string(output) != "leaf.o\n" {
		t.Fatalf("archive members: %v, %q", err, output)
	}
}

func containsRelocationKind(values []int, wanted int) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
