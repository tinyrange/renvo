package main

import (
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDumpVersionOneGoldenVectors(t *testing.T) {
	for _, name := range []string{"v1-core.hex", "v1-full.hex"} {
		name := name
		t.Run(name, func(t *testing.T) {
			// rtgunitc consumes binary units; decode the checked-in hexadecimal
			// vector through the conformance helper before exercising the CLI.
			binaryPath := filepath.Join(t.TempDir(), "input.rtgu")
			writeGoldenBinary(t, filepath.Join("..", "..", "rtgunit", "testdata", name), binaryPath)
			cmd := exec.Command("go", "run", ".", "-dump", binaryPath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("rtgunitc -dump: %v\n%s", err, output)
			}
			if string(output) != "package p\n" {
				t.Fatalf("dump output = %q", output)
			}
		})
	}
}

func writeGoldenBinary(t *testing.T, source string, destination string) {
	t.Helper()
	encoded, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	data, err := hex.DecodeString(strings.TrimSpace(string(encoded)))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, data, 0600); err != nil {
		t.Fatal(err)
	}
}
