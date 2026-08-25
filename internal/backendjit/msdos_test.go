package backendjit

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

func TestMSDOSCOMExample(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "examples", "msdos", "msdos_com.rtg")
	backend := New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
		t.TempDir(), backendcompiled.Backend{})
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "msdos/8086",
		"-s",
		"-o", "hello.com",
		filepath.Join(root, "examples", "msdos"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		backend)
	if !result.Ok {
		t.Fatalf("MS-DOS custom backend compile failed: %#v", result.Diagnostic)
	}
	image := result.Binary
	if len(image) == 0 || len(image) > 0xff00 {
		t.Fatalf("COM image size = %d, want 1..65280", len(image))
	}
	if len(image) >= 2 && image[0] == 'M' && image[1] == 'Z' {
		t.Fatal("COM image unexpectedly has an MZ header")
	}
	if !bytes.Contains(image, []byte("Hello from Renvo on MS-DOS!\r\n")) {
		t.Fatal("COM image does not contain the program data")
	}
	if bytes.Count(image, []byte{0xcd, 0x21}) < 2 {
		t.Fatal("COM image lacks DOS write and exit interrupts")
	}

	raw := backend.CompileSourceWithArena([]byte(`package main
func appMain() int { print("RAW PASS\n"); return 0 }
`), "msdos/8086", true, 4096)
	if !raw.Ok {
		t.Fatalf("raw backend source compile failed: %#v", raw.Diagnostic)
	}
	if !bytes.Contains(raw.Binary, []byte("RAW PASS\n")) {
		t.Fatal("raw backend source output does not contain program data")
	}

	cRoot := t.TempDir()
	cPath := filepath.Join(cRoot, "main.c")
	if err := os.WriteFile(cPath, []byte(`#include <stdio.h>
int main(void) {
	puts("C PASS");
	return 0;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cArgs := driver.NormalizeCCompilerCommand([]string{
		"renvo", "cc", "-backend", definition, "-t", "msdos/8086",
		"-s", "-o", "hello.com", cPath,
	})
	cResult := driver.CompileFromFS(cArgs[1:], root, filepath.Join(root, "std"), driver.OSFS{}, backend)
	if !cResult.Ok {
		t.Fatalf("MS-DOS C compile failed: %#v", cResult.Diagnostic)
	}
	if !bytes.Contains(cResult.Binary, []byte("C PASS")) {
		t.Fatal("C COM image does not contain program data")
	}
}
