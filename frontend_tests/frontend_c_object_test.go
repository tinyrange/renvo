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

func TestFrontendCObjectTypeLayoutSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectTypeLayoutSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectTypeLayoutSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectTypeLayoutSystemLink(t, selfHostedFrontendCompiler(t, root))
}

func TestFrontendCObjectControlFlowSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectControlFlowSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectControlFlowSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectControlFlowSystemLink(t, selfHostedFrontendCompiler(t, root))
}

func TestFrontendCObjectUnionBitfieldSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectUnionBitfieldSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectUnionBitfieldSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectUnionBitfieldSystemLink(t, selfHostedFrontendCompiler(t, root))
}

func TestFrontendCObjectTentativeDefinitionSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectTentativeDefinitionSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectTentativeDefinitionSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectTentativeDefinitionSystemLink(t, selfHostedFrontendCompiler(t, root))
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

func runCObjectUnionBitfieldSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "layout.c")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "layout.o")
	executable := filepath.Join(dir, "layout-test")
	if err := os.WriteFile(source, []byte(`union word {
	unsigned whole;
	unsigned char bytes[4];
};
struct flags {
	unsigned low : 3;
	unsigned high : 5;
	signed delta : 6;
	unsigned : 0;
	_Bool ready : 1;
	unsigned tail : 7;
};
int renvo_union_bits(void) {
	union word word;
	union word *word_cursor = &word;
	struct flags flags;
	struct flags *flag_cursor = &flags;
	word_cursor->whole = 0x04030201;
	flags.low = 5;
	flags.high = 17;
	flags.delta = -3;
	flags.ready = 1;
	flag_cursor->tail = 65;
	flags.low += 1;
	return sizeof(union word) + sizeof(struct flags) + word.bytes[0] +
		flags.low + flags.high + flags.delta + flags.ready + flags.tail;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`extern int renvo_union_bits(void);
int main(void) { return renvo_union_bits() == 103 ? 0 : 1; }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C union/bitfield object with Renvo: %v\n%s", err, combined)
	}
	link := exec.Command(linkerDriver, harness, object, "-o", executable)
	if combined, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link C union/bitfield object: %v\n%s", err, combined)
	}
	if combined, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked C union/bitfield object: %v, output %q", err, combined)
	}
}

func runCObjectTentativeDefinitionSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "storage.c")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "storage.o")
	executable := filepath.Join(dir, "storage-test")
	if err := os.WriteFile(source, []byte(`int renvo_accumulate(int value);
int shared;
extern int shared;
int shared;
static int hidden;
int singleton[];
int renvo_accumulate(int value) {
	extern int shared;
	static int calls;
	calls++;
	hidden++;
	shared += value;
	singleton[0]++;
	return shared + hidden + calls + singleton[0];
}
int renvo_accumulate(int value);
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`extern int renvo_accumulate(int);
int main(void) {
	return renvo_accumulate(20) == 23 && renvo_accumulate(20) == 46 ? 0 : 1;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C tentative-definition object with Renvo: %v\n%s", err, combined)
	}
	link := exec.Command(linkerDriver, harness, object, "-o", executable)
	if combined, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link C tentative-definition object: %v\n%s", err, combined)
	}
	if combined, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked C tentative-definition object: %v, output %q", err, combined)
	}
}

func runCObjectTypeLayoutSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}

	dir := t.TempDir()
	header := filepath.Join(dir, "layout.h")
	source := filepath.Join(dir, "layout.c")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "layout.o")
	executable := filepath.Join(dir, "layout-test")
	if err := os.WriteFile(header, []byte(`#ifndef RENVO_LAYOUT_H
#define RENVO_LAYOUT_H
#define RENVO_VALUE_COUNT 3
#define RENVO_EXPECTED_SIZE 24
typedef struct renvo_record {
	unsigned char tag;
	int values[RENVO_VALUE_COUNT];
	long tail;
} renvo_record;
enum renvo_mode {
	RENVO_MODE_BASE = 1 << 2,
	RENVO_MODE_NEXT,
	RENVO_MODE_MASK = RENVO_MODE_BASE | 2
};
typedef int (*renvo_binary_op)(int, int);
typedef struct renvo_callbacks {
	renvo_binary_op apply;
	int *values[3];
	int (*matrix)[3];
} renvo_callbacks;
#endif
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte(`#include "layout.h"
static renvo_record global_record;
static int global_value;
int renvo_layout(void) {
	renvo_record value;
	renvo_record *cursor = &value;
	int i = 0;
	int total = 0;
	cursor->tag = 7;
	value.values[0] = 1;
	value.values[1] = 2;
	value.values[2] = 3;
	value.tail = 13;
	global_record.tail = cursor->tail;
	for (i = 0; i < RENVO_VALUE_COUNT; i++) total += value.values[i];
	if (sizeof(renvo_record) != RENVO_EXPECTED_SIZE) return 1;
	if (_Alignof(renvo_record) != 8) return 2;
	if (__builtin_offsetof(renvo_record, tag) != 0) return 3;
	if (__builtin_offsetof(renvo_record, values) != 4) return 4;
	if (__builtin_offsetof(renvo_record, tail) != 16) return 5;
	if (sizeof(renvo_callbacks) != 40) return 10;
	if (__builtin_offsetof(renvo_callbacks, apply) != 0) return 11;
	if (__builtin_offsetof(renvo_callbacks, values) != 8) return 12;
	if (__builtin_offsetof(renvo_callbacks, matrix) != 32) return 13;
	if (RENVO_MODE_NEXT != 5 || RENVO_MODE_MASK != 6) return 14;
	if (cursor->tag != 7) return 6;
	if (total != 6) return 7;
	if (global_value != 0) return 8;
	if (global_record.tail != 13) return 9;
	return 5008;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`extern int renvo_layout(void);
int main(void) { return renvo_layout() == 5008 ? 0 : 1; }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C type/layout object with Renvo: %v\n%s", err, combined)
	}
	file, err := elf.Open(object)
	if err != nil {
		t.Fatalf("open C type/layout object: %v", err)
	}
	bss := file.Section(".bss")
	if bss == nil || bss.Type != elf.SHT_NOBITS || bss.Size < 32 {
		file.Close()
		t.Fatalf("C object BSS section = %#v", bss)
	}
	file.Close()
	link := exec.Command(linkerDriver, harness, object, "-o", executable)
	if combined, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link Renvo C type/layout object: %v\n%s", err, combined)
	}
	if combined, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked C type/layout object: %v, output %q", err, combined)
	}
}

func runCObjectControlFlowSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "control.c")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "control.o")
	executable := filepath.Join(dir, "control-test")
	if err := os.WriteFile(source, []byte(`int renvo_control(int choice) {
	int i = 0;
	int total = 0;
	do {
		i++;
		if (i < 3) continue;
		total += i;
	} while (i < 4);
	for (int j = 0; j < 3; j++) total += j;
	switch (choice) {
	case 0:
		total += 1;
	case 1:
		total += 2;
		break;
	default:
		total += 9;
	}
	goto done;
	total = 1000;
done:
	return total;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`extern int renvo_control(int);
int main(void) {
	return renvo_control(0) == 13 &&
		renvo_control(1) == 12 &&
		renvo_control(2) == 19 ? 0 : 1;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C control-flow object with Renvo: %v\n%s", err, combined)
	}
	link := exec.Command(linkerDriver, harness, object, "-o", executable)
	if combined, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link Renvo C control-flow object: %v\n%s", err, combined)
	}
	if combined, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked C control-flow object: %v, output %q", err, combined)
	}
}
