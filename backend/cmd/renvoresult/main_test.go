package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/backend/omnibus/resultabi"
)

func TestRunRequiresArtifactAndMemory(t *testing.T) {
	var output bytes.Buffer
	if err := run(nil, &output); err == nil {
		t.Fatal("missing inputs accepted")
	}
}

func TestHostedModePrintsOnlyPassAfterFullValidation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("ELF fixture")
	}
	compiler, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("C compiler not installed")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "result.c")
	artifact := filepath.Join(dir, "result.o")
	if err := os.WriteFile(source, []byte("unsigned char renvores[64];\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(compiler, "-std=c89", "-pedantic-errors", "-c", source, "-o", artifact)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile ELF fixture: %v\n%s", err, output)
	}
	address, err := resultabi.ELFSymbolAddress(artifact, resultabi.SymbolName)
	if err != nil {
		t.Fatal(err)
	}
	const profile = uint32(0x00320001)
	const signature = uint64(0x0123456789abcdef)
	block := resultabi.New(profile)
	block.Pass(signature)
	memory := filepath.Join(dir, "memory.bin")
	if err := os.WriteFile(memory, block[:], 0o644); err != nil {
		t.Fatal(err)
	}
	args := []string{
		"-artifact", artifact,
		"-memory", memory,
		"-base", fmt.Sprintf("%#x", address),
		"-expected-profile", fmt.Sprintf("%#x", profile),
		"-expected-signature", fmt.Sprintf("%#x", signature),
		"-hosted",
	}
	var stdout bytes.Buffer
	if err := run(args, &stdout); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "PASS\n" {
		t.Fatalf("hosted output = %q", got)
	}
	stdout.Reset()
	args[9] = "0x1"
	if err := run(args, &stdout); err == nil {
		t.Fatal("wrong signature accepted")
	}
	if stdout.Len() != 0 {
		t.Fatalf("failure wrote stdout %q", stdout.String())
	}
}
