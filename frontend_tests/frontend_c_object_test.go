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

func TestFrontendCObjectAggregateInitializerSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectAggregateInitializerSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectAggregateInitializerSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectAggregateInitializerSystemLink(t, selfHostedFrontendCompiler(t, root))
}

func TestFrontendCObjectVariadicSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectVariadicSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectVariadicSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectVariadicSystemLink(t, selfHostedFrontendCompiler(t, root))
}

func TestFrontendCObjectThreadStorageSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectThreadStorageSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectThreadStorageSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectThreadStorageSystemLink(t, selfHostedFrontendCompiler(t, root))
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

func runCObjectAggregateInitializerSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "aggregate.c")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "aggregate.o")
	executable := filepath.Join(dir, "aggregate-test")
	if err := os.WriteFile(source, []byte(`struct inner { int x; int y; };
struct record {
	int head;
	struct inner nested;
	int values[3];
	int tail;
};
union word { unsigned whole; unsigned char bytes[4]; };
struct packed { unsigned low : 3; unsigned high : 5; int value; };
struct text { char label[3]; unsigned char wrapped; };
typedef int (*binary_fn)(int, int);
static int expression_side;
static int expression_add(int left, int right) { return left + right; }
static int expression_mark(int value) { expression_side = expression_side * 10 + value; return value; }
static int ignored_variadic(int fixed, ...) { return fixed; }
int promotion_truth(void) { return (unsigned char)250 + 10 == 260; }
int renvo_aggregate(void) {
	struct record selected = {
		.tail = 5,
		.nested.y = 3,
		.nested.x = 2,
		.values[2] = 7,
		.values[0] = 1
	};
	struct record positional = { 1, { 2, 3 }, { 4, 5, 6 }, 7 };
	int sparse[5] = { [3] = 9, 10 };
	int inferred[] = { [2] = 8, 9 };
	char string_inferred[] = "A" "B";
	char string_exact[2] = "ok";
	unsigned char string_padded[4] = u8"x";
	int overwritten[2] = { [0] = 1, [0] = 4 };
	union word word = { .whole = 0x04030201 };
	union word bytes = (union word){ .bytes = { 9, 8, 7, 6 } };
	struct packed packed = { .high = 17, .low = 5, .value = 4 };
	struct packed sequence = { 6, 3, 2 };
	struct text text = { "ok", 300 };
	struct text named = { .wrapped = 301, .label = "hi" };
	const int qualified = 2;
	volatile int observed = qualified;
	unsigned char promotion_left = 250, promotion_right = 10;
	short promotion_wide = 30000;
	signed char promotion_negative = -1;
	unsigned int promotion_one = 1;
	long promotion_large = -1;
	_Bool promotion_bool = 260;
	int promotion_selected = promotion_negative < promotion_one ? 7 : 9;
	binary_fn operation = expression_add;
	int lazy = 1 ? expression_mark(2) : expression_mark(3);
	int comma = (expression_mark(4), expression_mark(5));
	if (promotion_left + promotion_right != 260 || promotion_wide + promotion_wide != 60000 ||
		promotion_negative < promotion_one || !(promotion_large < promotion_one)) return -1;
	if (!promotion_truth()) return -2;
	if (promotion_bool != 1) return -3;
	if (promotion_selected != 9) return -4;
	if (lazy != 2 || comma != 5 || expression_side != 245) return -5;
	if (operation(7, 5) != 12 || ~promotion_left != -251) return -6;
	if (sizeof inferred != 16 || ignored_variadic(9, expression_mark(6), 2.5) != 9 || expression_side != 2456) return -7;
	if (sizeof string_inferred != 3 || string_inferred[0] != 'A' || string_inferred[2] != 0 ||
		string_exact[1] != 'k' || string_padded[0] != 'x' || string_padded[1] != 0 || string_padded[3] != 0 || sizeof "xy" != 3) return -8;
	if (text.label[0] != 'o' || text.label[2] != 0 || text.wrapped != 44) return -9;
	if (named.label[1] != 'i' || named.label[2] != 0 || named.wrapped != 45 || ((int[2]){4, 5})[1] != 5) return -10;
	return selected.tail + selected.nested.x + selected.values[2] +
		positional.nested.y + sparse[4] + inferred[3] + ((struct inner){ .x = 4, .y = 6 }).y +
		word.bytes[0] + bytes.bytes[1] + packed.high + packed.low + packed.value +
		sequence.low + sequence.high + sequence.value + overwritten[0] + observed;
}

`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`extern int renvo_aggregate(void);
int main(void) { return renvo_aggregate() == 94 ? 0 : 1; }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C aggregate-initializer object with Renvo: %v\n%s", err, combined)
	}
	link := exec.Command(linkerDriver, harness, object, "-o", executable)
	if combined, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link C aggregate-initializer object: %v\n%s", err, combined)
	}
	if combined, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked C aggregate-initializer object: %v, output %q", err, combined)
	}
}

func runCObjectVariadicSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "variadic.c")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "variadic.o")
	executable := filepath.Join(dir, "variadic-test")
	if err := os.WriteFile(source, []byte(`int printf(const char *format, ...);
int renvo_variadic(void) {
	unsigned char small = 7;
	short negative = -3;
	return printf("renvo variadic %d %d %.1f\n", small, negative, (float)2.5);
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`extern int renvo_variadic(void);
int main(void) { return renvo_variadic() == 24 ? 0 : 1; }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C variadic object with Renvo: %v\n%s", err, combined)
	}
	link := exec.Command(linkerDriver, harness, object, "-o", executable)
	if combined, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link C variadic object: %v\n%s", err, combined)
	}
	combined, err := exec.Command(executable).CombinedOutput()
	if err != nil {
		t.Fatalf("run linked C variadic object: %v, output %q", err, combined)
	}
	if string(combined) != "renvo variadic 7 -3 2.5\n" {
		t.Fatalf("variadic output = %q", combined)
	}
}

func runCObjectThreadStorageSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "thread.c")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "thread.o")
	executable := filepath.Join(dir, "thread-test")
	if err := os.WriteFile(source, []byte(`extern _Thread_local int counter;
int renvo_tls_step(int delta) { counter += delta; return counter; }
_Thread_local int counter;
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`#include <pthread.h>
extern int renvo_tls_step(int);
struct result { int first; int second; };
static void *run(void *opaque) {
	struct result *result = opaque;
	result->first = renvo_tls_step(1);
	result->second = renvo_tls_step(2);
	return 0;
}
int main(void) {
	pthread_t left, right;
	struct result a = {0}, b = {0};
	if (pthread_create(&left, 0, run, &a) || pthread_create(&right, 0, run, &b)) return 1;
	pthread_join(left, 0); pthread_join(right, 0);
	return a.first == 1 && a.second == 3 && b.first == 1 && b.second == 3 && renvo_tls_step(5) == 5 ? 0 : 2;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C thread-storage object with Renvo: %v\n%s", err, combined)
	}
	link := exec.Command(linkerDriver, harness, object, "-pthread", "-o", executable)
	if combined, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link C thread-storage object: %v\n%s", err, combined)
	}
	if combined, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked C thread-storage object: %v, output %q", err, combined)
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
unsigned long renvo_choose_type(unsigned long input) {
	__typeof__(__builtin_choose_expr(
		sizeof(input) <= sizeof(int), (unsigned int)0, (unsigned long)0)) value = input;
	return value;
}
unsigned long renvo_shift_precedence(unsigned long align) {
	return (1UL << 8 * align) - 1;
}
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
	if (renvo_choose_type(0x123456789abcdef0UL) != 0x123456789abcdef0UL) return 15;
	if (renvo_shift_precedence(7) != 0x00ffffffffffffffUL) return 16;
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
	unsigned char narrow = choice;
	switch (narrow) { case 300: total = 1000; default: break; }
	unsigned mask = ~0U;
	switch (mask) { case -1: total += 4; break; }
	goto done;
	total = 1000;
done:
	return total;
}
int renvo_nested_label(int value) {
	if (!value) {
fail:
		return 7;
	}
	if (value == 2) goto fail;
	return 9;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`extern int renvo_control(int);
extern int renvo_nested_label(int);
int main(void) {
	return renvo_control(0) == 17 &&
		renvo_control(1) == 16 &&
		renvo_control(2) == 23 &&
		renvo_nested_label(0) == 7 &&
		renvo_nested_label(1) == 9 &&
		renvo_nested_label(2) == 7 ? 0 : 1;
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
