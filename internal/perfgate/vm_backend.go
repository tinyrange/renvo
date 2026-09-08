package perfgate

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Prepare the same browser-style custom backend from each source revision.
// Frontend lowering and backend preparation are outside this VM-only workload;
// the measured input is the canonical unit, never Go source.
func (h harness) prepareVMBackend(b *build, frontend string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.policy.InvocationTimeoutSeconds)*time.Second)
	defer cancel()
	run := func(command string, args ...string) error {
		cmd := exec.CommandContext(ctx, command, args...)
		cmd.Dir, cmd.Env = b.root, compilerEnv(b.root)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("prepare VM backend: %w\n%s", err, out)
		}
		return nil
	}
	jit := filepath.Join(b.directory, "backendjit")
	if err := run("go", "build", "-o", jit, "./cmd/renvowasibackendjit"); err != nil {
		return err
	}
	source, err := os.ReadFile(filepath.Join(b.root, "backend/definitions/wasm32.rtg"))
	if err != nil {
		return err
	}
	definition := filepath.Join(b.directory, "backend.rtg")
	text := strings.Replace(string(source), "target vm/vm32 {", "target performance/vm32 {", 1)
	if text == string(source) {
		return fmt.Errorf("VM target missing from backend definition")
	}
	if err := os.WriteFile(definition, []byte(text), 0600); err != nil {
		return err
	}
	b.stage2 = filepath.Join(b.directory, "backend.rnvb")
	if err := run(jit, "-definition", definition, "-target", "performance/vm32", "-o", b.stage2); err != nil {
		return err
	}
	sandbox := filepath.Join(b.root, "sandbox")
	if err := os.MkdirAll(sandbox, 0755); err != nil {
		return err
	}
	fixture, err := os.MkdirTemp(sandbox, "vm-gate-fixture-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(fixture)
	program := filepath.Join(fixture, "main.go")
	if err := os.WriteFile(program, h.fixture, 0600); err != nil {
		return err
	}
	b.unit = filepath.Join(b.directory, "input.unit")
	fmt.Fprintf(h.log, "%s: VM workload source SHA256=%x\n", filepath.Base(b.directory), sha256.Sum256(h.fixture))
	return run(frontend, "-backend", definition, "-t", "performance/vm32", "-emit-unit", "-o", b.unit, program)
}

func (h harness) sampleVMBackend(b *build, name string) (Sample, error) {
	output := filepath.Join(b.directory, name+".rnvb")
	sample, err := h.execute(b, b.stage2, []string{"-t", "performance/vm32", "-s", "-arena-size", "16777216", "-o", output, b.unit}, output)
	if err != nil {
		return sample, err
	}
	// The artifact gate still protects the VM-hosted compiler, not the much
	// smaller program it produces. The generated program must run successfully.
	info, err := os.Stat(b.stage2)
	if err != nil {
		return sample, err
	}
	sample.Artifact = uint64(info.Size())
	return sample, h.smoke(b, output)
}
