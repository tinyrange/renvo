package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendCObjectExportArgumentOrderSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectExportArgumentOrderSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectExportArgumentOrderSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectExportArgumentOrderSystemLink(t, selfHostedFrontendCompiler(t, root))
}

func runCObjectExportArgumentOrderSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "argument-order.c")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "argument-order.o")
	executable := filepath.Join(dir, "argument-order-test")
	if err := os.WriteFile(source, []byte(`
long encode_arguments(long first, long second, long third) {
	return first * 100 + second * 10 + third;
}
struct record { long value; };
static void initialize_record(struct record *, const char *, long *);
long forward_defined_string_argument(void) {
	static long key = 7;
	struct record record = {0};
	initialize_record(&record, "record", &key);
	return record.value;
}
static void initialize_record(struct record *record, const char *name, long *key) {
	record->value = name[0] + *key;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`
#include <stdio.h>
extern long encode_arguments(long, long, long);
extern long forward_defined_string_argument(void);
int main(void) {
	if (encode_arguments(1, 2, 3) != 123 || forward_defined_string_argument() != 'r' + 7)
		return 1;
	puts("PASS");
	return 0;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C argument-order object with Renvo: %v\n%s", err, output)
	}
	link := exec.Command(linkerDriver, harness, object, "-o", executable)
	if output, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link C argument-order object: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil || string(output) != "PASS\n" {
		t.Fatalf("run linked C argument-order object: %v, output %q", err, output)
	}
}
