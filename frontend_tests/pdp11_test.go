package frontend_tests

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestPDP11Tab5Build(t *testing.T) {
	for _, source := range []string{"main.c", "tests/main.c"} {
		t.Run(source, func(t *testing.T) { buildPDP11Tab5(t, source) })
	}
}

func buildPDP11Tab5(t *testing.T, source string) {
	t.Helper()
	root := repoRoot(t)
	frontend := integratedFrontendCompiler(t, root)
	output := filepath.Join(t.TempDir(), "tab5-pdp11.elf")
	cmd := frontendCommand(frontend, "cc", "-backend", filepath.Join(root, "backends/esp32p4.rtg"), "-t", "esp32p4/riscv32", "-tags", "m5tab5", "-o", output, filepath.Join(root, "examples/device/tab5_pdp11", source))
	cmd.Dir = root
	cmd.Env = frontendCommandEnv(frontend.env, root)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Tab5 compile: %v\n%s", err, out)
	}
	file, err := elf.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if file.Class != elf.ELFCLASS32 || file.Machine != elf.EM_RISCV {
		t.Fatalf("unexpected target: %v/%v", file.Class, file.Machine)
	}
	// The freestanding writer describes BSS with a zero-file-size PT_LOAD,
	// not a named section. Check the actual loader memory reservation.
	foundBSS := false
	for _, segment := range file.Progs {
		if segment.Type == elf.PT_LOAD && segment.Filesz == 0 && segment.Memsz > 0 {
			foundBSS = true
			if segment.Vaddr != 0x4ff00100 || segment.Vaddr+segment.Memsz > 0x4ff2a000 {
				t.Fatalf("BSS overlaps target scratch/stack or guest SRAM: %+v", segment.ProgHeader)
			}
		}
	}
	if !foundBSS {
		t.Fatal("missing BSS load segment")
	}
}

func TestPDP11Core(t *testing.T) {
	root := repoRoot(t)
	for _, stage := range []string{"stage0", "stage3"} {
		t.Run(stage, func(t *testing.T) {
			frontend := frontendCompiler(t, root)
			if stage == "stage3" {
				frontend = selfHostedFrontendCompiler(t, root)
			}
			binary := filepath.Join(t.TempDir(), "pdp11-core")
			cmd := frontendCommand(frontend, "cc", "-t", frontend.target, "-o", binary, filepath.Join(root, "examples/pdp11/tests/core.c"))
			cmd.Dir = root
			cmd.Env = cExecutableFrontendEnv(frontend, root, root)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("compile: %v\n%s", err, out)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, binary).CombinedOutput()
			if err != nil || string(out) != "PASS\n" {
				t.Fatalf("core: %v\n%s", err, out)
			}
		})
	}
}

// Boot images remain external: normal presubmit must neither fetch nor require
// historical software. This opt-in test verifies a real Renvo-built V7 boot.
func TestPDP11UnixV7Boot(t *testing.T) {
	image := os.Getenv("RENVO_PDP11_V7_IMAGE")
	if image == "" {
		t.Skip("set RENVO_PDP11_V7_IMAGE to a TUHS v7_rk05_1145 image")
	}
	original, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	before := sha256.Sum256(original)
	root := repoRoot(t)
	frontend := integratedFrontendCompiler(t, root)
	frontend.target = frontendTarget(t)
	binary := filepath.Join(t.TempDir(), "pdp11")
	cmd := frontendCommand(frontend, "cc", "-t", frontend.target, "-o", binary, filepath.Join(root, "examples/pdp11/main.c"))
	cmd.Dir = root
	cmd.Env = cExecutableFrontendEnv(frontend, root, root)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, out)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, binary, "--headless", "--ticks", "10000", "--script", filepath.Join(root, "examples/pdp11/tests/v7.script"), image)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("boot: %v\n%s", err, out)
	}
	// Check actual command output, not just terminal-echoed input.
	if !bytes.Contains(out, []byte("\r\nRENVO_V7_BOOT_OK\r\n")) || !bytes.Contains(out, []byte("RKUNIX\r\n")) {
		t.Fatalf("missing V7 shell output:\n%s", out)
	}
	if !bytes.Contains(out, []byte("\r\nRENVO_DISK_WRITE_OK\r\n")) {
		t.Fatalf("missing guest file readback:\n%s", out)
	}
	after, err := os.ReadFile(image)
	if err != nil || sha256.Sum256(after) != before {
		t.Fatalf("original image changed: %v", err)
	}
}
