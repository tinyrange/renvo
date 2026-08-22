package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendCObjectAddressMutationSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectAddressMutationSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectAddressMutationSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectAddressMutationSystemLink(t, selfHostedFrontendCompiler(t, root))
}

func runCObjectAddressMutationSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "address-mutation.c")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "address-mutation.o")
	executable := filepath.Join(dir, "address-mutation-test")
	if err := os.WriteFile(source, []byte(`
#include <stdint.h>
struct item { uint64_t value; } __attribute__((packed));
struct item items[4];
void fill_items(void) {
	uint32_t index = 0;
	uint32_t step;
	for (step = 0; step < 3; step++) {
		items[index].value = step + 1;
		if (++index >= 4) break;
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`
#include <stdint.h>
struct item { uint64_t value; } __attribute__((packed));
extern struct item items[4];
extern void fill_items(void);
int main(void) {
	fill_items();
	return items[0].value == 1 && items[1].value == 2 &&
		items[2].value == 3 && items[3].value == 0 ? 0 : 1;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C address-mutation object with Renvo: %v\n%s", err, output)
	}
	link := exec.Command(linkerDriver, harness, object, "-o", executable)
	if output, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link C address-mutation object: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked C address-mutation object: %v, output %q", err, output)
	}
}
