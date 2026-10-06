package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"renvo.dev/internal/linkedimage"
	"testing"
)

func TestDefinitionCompilerSelfHosts(t *testing.T) {
	root := repoRoot(t)
	frontend := selfHostedFrontendCompiler(t, root)
	output := filepath.Join(t.TempDir(), "rtg-selfhost")
	command := frontendCommand(frontend,
		"-t", frontend.target, "-s", "-o", output,
		"./frontend_tests/testdata/rtg_selfhost")
	command.Dir = root
	command.Env = frontendCommandEnv(frontend.env, root)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile definition compiler with Renvo: %v\n%s", err, combined)
	}
	source, err := exec.Command(output, "generate").CombinedOutput()
	if err != nil {
		t.Fatalf("self-host evaluator generation: %v\n%s", err, source)
	}
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "evaluator.go")
	imagePath := filepath.Join(directory, "evaluator.rnvi")
	bytecodePath := filepath.Join(directory, "evaluator.rnvm")
	if err = os.WriteFile(sourcePath, source, 0600); err != nil {
		t.Fatal(err)
	}
	command = frontendCommand(frontend, "-t", "vm/vm32", "-s", "-emit-image", "-arena-size", "262144", "-o", imagePath, sourcePath)
	command.Dir = root
	command.Env = frontendCommandEnv(frontend.env, root)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("self-host compile generated evaluator: %v\n%s", err, combined)
	}
	image, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	_, format, bytecode, ok := linkedimage.Payload(image)
	if !ok || format != linkedimage.FormatBytecode {
		t.Fatal("invalid evaluator image")
	}
	if err = os.WriteFile(bytecodePath, bytecode, 0600); err != nil {
		t.Fatal(err)
	}
	combined, err := exec.Command(output, sourcePath, bytecodePath).CombinedOutput()
	if err != nil {
		t.Fatalf("self-host materialization: %v\n%s", err, combined)
	}
	if string(combined) != "PASS\n" {
		t.Fatalf("self-host materialization output = %q", combined)
	}
}
