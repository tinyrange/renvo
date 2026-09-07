package frontend_tests

import (
	"debug/elf"
	"path/filepath"
	"testing"
)

func TestFrontendRollerCANNanoC6(t *testing.T) {
	root := repoRoot(t)
	frontend := integratedFrontendCompiler(t, root)
	output := filepath.Join(t.TempDir(), "rollercan.elf")
	command := frontendCommand(frontend,
		"-backend", filepath.Join(root, "backends", "esp32c6.rtg"),
		"-t", "esp32c6/riscv32", "-tags", "m5nanoc6", "-s", "-o", output,
		filepath.Join(root, "examples", "device", "rollercan"))
	command.Dir = root
	command.Env = frontendCommandEnv(frontend.env, root)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile RollerCAN example: %v\n%s", err, combined)
	}
	file, err := elf.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if file.Class != elf.ELFCLASS32 || file.Machine != elf.EM_RISCV {
		t.Fatalf("output = %v/%v, want ELF32/RISC-V", file.Class, file.Machine)
	}
}
