package frontend_tests

import (
	"os"
	"path/filepath"
	"testing"

	wireunit "renvo.dev/internal/unit"
)

func TestFrontendBuildsMixedESP32C6Blink(t *testing.T) {
	root := repoRoot(t)
	frontend := frontendCompiler(t, root)
	if frontend.compiler == "" {
		t.Skip("frontend compiler unavailable")
	}
	output := filepath.Join(t.TempDir(), "blink-mixed.unit")
	command := frontendCommand(frontend,
		"-backend", filepath.Join(root, "backends", "esp32c6.rtg"),
		"-emit-unit", "-t", "esp32c6/riscv32", "-tags", "m5nanoc6", "-o", output,
		"./examples/device/blink_mixed")
	command.Dir = root
	command.Env = frontendCommandEnv(frontend.env, root)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile mixed Go/C ESP32-C6 blink: %v\n%s", err, combined)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	binding, ok := wireunit.ReadTargetBinding(data)
	if !ok || binding.Target != "esp32c6/riscv32" {
		t.Fatalf("mixed blink target binding = %#v, ok=%v", binding, ok)
	}
}
