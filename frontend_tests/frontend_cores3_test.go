package frontend_tests

import (
	"debug/elf"
	"path/filepath"
	"testing"
)

func TestFrontendCoreS3SEForms(t *testing.T) {
	root := repoRoot(t)
	frontend := integratedFrontendCompiler(t, root)
	output := filepath.Join(t.TempDir(), "cores3_forms.elf")
	command := frontendCommand(frontend,
		"-backend", filepath.Join(root, "backends", "esp32s3.rtg"),
		"-t", "esp32s3/xtensa_lx7", "-tags", "m5cores3se", "-s", "-o", output,
		filepath.Join(root, "examples", "device", "cores3_forms"))
	command.Dir = root
	command.Env = frontendCommandEnv(frontend.env, root)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile CoreS3-SE forms example: %v\n%s", err, combined)
	}
	file, err := elf.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if file.Class != elf.ELFCLASS32 || file.Machine != elf.EM_XTENSA {
		t.Fatalf("output = %v/%v, want ELF32/Xtensa", file.Class, file.Machine)
	}
}
