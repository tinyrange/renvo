package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendCObjectAggregatePointerRelocation(t *testing.T) {
	root := repoRoot(t)
	runCObjectAggregatePointerRelocation(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectAggregatePointerRelocation(t *testing.T) {
	root := repoRoot(t)
	runCObjectAggregatePointerRelocation(t, selfHostedFrontendCompiler(t, root))
}

func runCObjectAggregatePointerRelocation(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the first C object target is Linux/amd64")
	}
	linker, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "relocation.c")
	object := filepath.Join(dir, "relocation.o")
	executable := filepath.Join(dir, "relocation")
	if err := os.WriteFile(source, []byte(`
struct image { unsigned char *data; };
struct nested { unsigned char padding[3]; unsigned char value; };
struct image_table { struct nested nested; };
struct callbacks { int (*compare)(const char *, const char *); };
extern int strcmp(const char *, const char *);
unsigned char raw_data[8] = { 42 };
struct image image = { .data = raw_data };
struct image_table image_table = { .nested.value = 43 };
unsigned char *nested_value = &image_table.nested.value;
struct callbacks callbacks = { .compare = strcmp };
int main(void) {
	return image.data == raw_data && image.data[0] == 42 &&
		nested_value == &image_table.nested.value && *nested_value == 43 &&
		callbacks.compare("renvo", "renvo") == 0 ? 0 : 1;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile aggregate pointer relocation: %v\n%s", err, output)
	}
	link := exec.Command(linker, object, "-o", executable)
	if output, err := link.CombinedOutput(); err != nil {
		t.Fatalf("link aggregate pointer relocation: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run aggregate pointer relocation: %v\n%s", err, output)
	}
}
