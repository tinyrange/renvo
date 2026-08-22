package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendCAsmChecksumSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCAsmChecksumSystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CAsmChecksumSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCAsmChecksumSystemLink(t, selfHostedFrontendCompiler(t, root))
}

func runCAsmChecksumSystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the C object target is Linux/amd64")
	}
	linkerDriver, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "checksum.c")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "checksum.o")
	executable := filepath.Join(dir, "checksum-test")
	if err := os.WriteFile(source, []byte(`
unsigned short renvo_csum_fold(unsigned int sum) {
	asm("addl %1,%0\n\tadcl $0xffff,%0"
		: "=r"(sum) : "r"(sum << 16), "0"(sum & 0xffff0000));
	return (unsigned short)(~sum >> 16);
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`
#include <stdint.h>
extern unsigned short renvo_csum_fold(unsigned int);
static uint16_t reference(uint32_t value) {
	uint64_t wide = (uint64_t)(value & 0xffff0000U) + (uint32_t)(value << 16);
	uint32_t folded = (uint32_t)(wide + 0xffffU + (wide >> 32));
	return (uint16_t)(~folded >> 16);
}
int main(void) {
	static const uint32_t values[] = {
		0, 1, 0xffffffffU, 0x12345678U, 0xffff0001U, 0x80008000U, 0xdeadbeefU,
	};
	for (unsigned int i = 0; i < sizeof(values) / sizeof(values[0]); i++)
		if (renvo_csum_fold(values[i]) != reference(values[i])) return 1;
	return 0;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend, "cc", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile C checksum object with Renvo: %v\n%s", err, output)
	}
	link := exec.Command(linkerDriver, harness, object, "-o", executable)
	if output, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link C checksum object: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked C checksum object: %v, output %q", err, output)
	}
}
