package frontend_tests

import (
	"debug/elf"
	"os"
	"path/filepath"
	"testing"
)

func TestFrontendCPragmaGoMicrocontrollerExamples(t *testing.T) {
	root := repoRoot(t)
	frontend := integratedFrontendCompiler(t, root)
	for _, example := range []string{"blink_c", "button_rgb_c"} {
		t.Run(example, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), example+".elf")
			command := frontendCommand(frontend,
				"cc", "-backend", filepath.Join(root, "backends", "esp32c6.rtg"),
				"-t", "esp32c6/riscv32", "-tags", "m5nanoc6", "-s", "-o", output,
				filepath.Join(root, "examples", "device", example, "main.c"))
			command.Dir = root
			command.Env = frontendCommandEnv(frontend.env, root)
			if combined, err := command.CombinedOutput(); err != nil {
				t.Fatalf("compile %s: %v\n%s", example, err, combined)
			}
			if info, err := os.Stat(output); err != nil || info.Size() == 0 {
				t.Fatalf("%s output: info=%v err=%v", example, info, err)
			}
			file, err := elf.Open(output)
			if err != nil {
				t.Fatalf("open %s output: %v", example, err)
			}
			defer file.Close()
			if file.Class != elf.ELFCLASS32 || file.Machine != elf.EM_RISCV {
				t.Fatalf("%s output = %v/%v, want ELF32/RISC-V", example, file.Class, file.Machine)
			}
		})
	}
}
