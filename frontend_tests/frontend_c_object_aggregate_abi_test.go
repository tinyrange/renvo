package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendCObjectAggregateABISystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectAggregateABISystemLink(t, frontendCompiler(t, root), "")
}

func TestFrontendStage3CObjectAggregateABISystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectAggregateABISystemLink(t, selfHostedFrontendCompiler(t, root), "")
}

func TestFrontendCCustomRTGObjectAggregateABISystemLink(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	root := repoRoot(t)
	runCObjectAggregateABISystemLink(t, integratedFrontendCompiler(t, root), root,
		"-backend", filepath.Join(root, "backends", "linux_amd64_object.rtg"),
		"-t", "linux-object/amd64")
}

func runCObjectAggregateABISystemLink(
	t *testing.T, frontend frontendConfig, compileDir string, backendArgs ...string,
) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "aggregate-abi.c")
	harness := filepath.Join(dir, "harness.c")
	provider := filepath.Join(dir, "provider.c")
	object := filepath.Join(dir, "aggregate-abi.o")
	providerObject := filepath.Join(dir, "provider.o")
	executable := filepath.Join(dir, "aggregate-abi-test")
	if err := os.WriteFile(source, []byte(`
struct pair { long first; long second; };
struct pair add_pair(struct pair left, struct pair right) {
	struct pair result = {
		left.first + right.first,
		left.second + right.second,
	};
	return result;
}
struct empty {};
struct holder { struct empty lock; unsigned long value; };
unsigned long preserve_after_empty_assignment(void) {
	struct holder holder = { .value = 0x3f8 };
	holder.lock = (struct empty){};
	return holder.value;
}
typedef struct { unsigned long value; } protection_t;
static unsigned long protection_value(protection_t protection) {
	return protection.value;
}
#define make_protection(value) ((protection_t) { (value) })
unsigned long preserve_compound_literal_assignment(protection_t protection) {
	protection = make_protection(protection_value(protection));
	return protection.value;
}
extern protection_t external_protection(protection_t, void *);
unsigned long call_external_protection(unsigned long value) {
	return external_protection((protection_t){value}, (void *)0).value;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(provider, []byte(`
typedef struct { unsigned long value; } protection_t;
protection_t external_protection(protection_t protection, void *context) {
	(void)context;
	protection.value += 1;
	return protection;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`
#include <stdio.h>
struct pair { long first; long second; };
extern struct pair add_pair(struct pair, struct pair);
extern unsigned long preserve_after_empty_assignment(void);
typedef struct { unsigned long value; } protection_t;
extern unsigned long preserve_compound_literal_assignment(protection_t);
extern unsigned long call_external_protection(unsigned long);
int main(void) {
	struct pair result = add_pair((struct pair){1, 2}, (struct pair){10, 20});
	if (result.first != 11 || result.second != 22 ||
	    preserve_after_empty_assignment() != 0x3f8 ||
	    preserve_compound_literal_assignment((protection_t){2}) != 2 ||
	    call_external_protection(41) != 42)
		return 1;
	puts("PASS");
	return 0;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	compileArgs := []string{"cc"}
	compileArgs = append(compileArgs, backendArgs...)
	compileArgs = append(compileArgs, "-c", source, "-o", object)
	command := frontendCommand(frontend, compileArgs...)
	command.Dir = dir
	if compileDir != "" {
		command.Dir = compileDir
	}
	command.Env = frontendCommandEnv(frontend.env, dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C aggregate-ABI object with Renvo: %v\n%s", err, output)
	}
	compileProvider := exec.Command(linkerDriver, "-c", provider, "-o", providerObject)
	if output, err := compileProvider.CombinedOutput(); err != nil {
		t.Fatalf("compile system aggregate-ABI provider: %v\n%s", err, output)
	}
	link := exec.Command(linkerDriver, harness, object, providerObject, "-o", executable)
	if output, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link C aggregate-ABI object: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil || string(output) != "PASS\n" {
		t.Fatalf("run linked C aggregate-ABI object: %v, output %q", err, output)
	}
}
