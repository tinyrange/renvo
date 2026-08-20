package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendCObjectCompoundAssignmentSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectCompoundAssignmentSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectCompoundAssignmentSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectCompoundAssignmentSystemLink(t, selfHostedFrontendCompiler(t, root))
}

func runCObjectCompoundAssignmentSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "compound-assignment.c")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "compound-assignment.o")
	executable := filepath.Join(dir, "compound-assignment-test")
	if err := os.WriteFile(source, []byte(`
#include <stdint.h>
struct range { uint64_t size; };
struct identifier { char bytes[8]; };
struct over_aligned { uint64_t words[7]; } __attribute__((aligned(16)));
uint64_t shrink(uint64_t size, uint64_t base, uint64_t region_base) {
	size -= base - region_base;
	return size;
}
uint64_t shrink_member(struct range *range, uint64_t base, uint64_t region_base) {
	range->size -= base - region_base;
	return range->size;
}
int first_byte(struct identifier *identifier) { return *identifier->bytes; }
uint64_t aligned_distance(struct over_aligned *first, struct over_aligned *last) {
	return last - first;
}
static uint64_t ordered_pair(uint64_t first, uint64_t second) {
	return first * 10 + second;
}
uint64_t constant_argument_order(void) { return ordered_pair(7, 2); }
static uint64_t variable_pair(uint64_t first, uint64_t second) {
	return first * 10 + second;
}
uint64_t mixed_argument_order(uint64_t value) { return variable_pair(value, 2); }
static int allocation_mm;
static uint64_t allocate_page(uint32_t flags, uint32_t order) {
	return (uint64_t)flags * 10 + order;
}
static uint64_t allocation_shape(int *mm, uint64_t address) {
	uint32_t flags = 100;
	if (mm == &allocation_mm)
		flags = 7;
	flags &= ~2U;
	return allocate_page(flags, 0);
}
uint64_t allocation_argument_order(void) { return allocation_shape(0, 123); }
static uint64_t loop_values[3] = {11, 17, 23};
static uint64_t loop_lookup(int index) { return loop_values[index]; }
uint64_t loop_varying_single_call(void) {
	uint64_t result = 0;
	for (int index = 0; index < 3; index++)
		result += loop_lookup(index);
	return result;
}
static void initialize_literal(uint64_t *value, const char *name, uint64_t *key) {
	*value = (uint64_t)(unsigned char)*name + *key;
}
uint64_t internal_string_literal_argument(void) {
	uint64_t values[2] = {0, 0};
	uint64_t key = 3;
	for (int index = 0; index < 2; index++)
		initialize_literal(&values[index], "A", &key);
	return values[0] + values[1];
}
struct bitmask { uint64_t bits[1]; };
typedef struct bitmask mask_var[1];
struct hash_attrs { mask_var mask; };
uint64_t nested_array_member_pointer_decay(void) {
	struct hash_attrs attrs;
	attrs.mask[0].bits[0] = 73;
	return *(uint64_t *)(void *)((attrs.mask)->bits);
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`
#include <stdint.h>
struct range { uint64_t size; };
struct identifier { char bytes[8]; };
struct over_aligned { uint64_t words[7]; } __attribute__((aligned(16)));
extern uint64_t shrink(uint64_t, uint64_t, uint64_t);
extern uint64_t shrink_member(struct range *, uint64_t, uint64_t);
extern int first_byte(struct identifier *);
extern uint64_t aligned_distance(struct over_aligned *, struct over_aligned *);
extern uint64_t constant_argument_order(void);
extern uint64_t mixed_argument_order(uint64_t);
extern uint64_t allocation_argument_order(void);
extern uint64_t loop_varying_single_call(void);
extern uint64_t internal_string_literal_argument(void);
extern uint64_t nested_array_member_pointer_decay(void);
int main(void) {
	struct range range = {100};
	struct identifier identifier = {{37}};
	struct over_aligned aligned[2];
	return shrink(100, 40, 10) == 70 &&
		shrink_member(&range, 40, 10) == 70 && range.size == 70 &&
		first_byte(&identifier) == 37 && sizeof(struct over_aligned) == 64 &&
		aligned_distance(&aligned[0], &aligned[1]) == 1 &&
		constant_argument_order() == 72 && mixed_argument_order(7) == 72 &&
		allocation_argument_order() == 1000 &&
		loop_varying_single_call() == 51 &&
		internal_string_literal_argument() == 136 &&
		nested_array_member_pointer_decay() == 73 ? 0 : 1;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C compound-assignment object with Renvo: %v\n%s", err, output)
	}
	link := exec.Command(linkerDriver, harness, object, "-o", executable)
	if output, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link C compound-assignment object: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked C compound-assignment object: %v, output %q", err, output)
	}
}
