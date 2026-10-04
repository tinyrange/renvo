//go:build !renvo

package backendjit

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

// This existing corpus program exercises the raw syscall intrinsic rather
// than read/write wrappers. It must survive prepared lowering as well.
func TestPreparedRawSyscallProtocols(t *testing.T) {
	if hostTarget() == "" {
		t.Skip("no in-process prepared backend")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"darwin/arm64", "openbsd/amd64"} {
		t.Run(target, func(t *testing.T) {
			definition := copyNativeDefinition(t, root, "target "+target+" {", "target custom/syscalls {")
			stdRoot := filepath.Join(root, "std")
			backend := New(definition, filepath.Join(root, "backend"), stdRoot, t.TempDir(), backendcompiled.Backend{})
			compile := func(source string) driver.CompileResult {
				return driver.CompileFromFS([]string{"-backend", definition, "-t", "custom/syscalls", "-s", "-o", "app", source}, root, stdRoot, driver.OSFS{}, backend)
			}
			source := filepath.Join(root, "backend", "tests", "darwin_getdirentries_intrinsic.go")
			result := compile(source)
			if !result.Ok {
				t.Fatalf("prepared raw syscall: %#v", result.Diagnostic)
			}
			if target == "darwin/arm64" {
				if !bytes.Contains(result.Binary, []byte("_getdirentries\x00")) {
					t.Fatal("directory intrinsic did not emit its declared libc adapter")
				}
			} else {
				image, err := elf.NewFile(bytes.NewReader(result.Binary))
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, program := range image.Progs {
					if uint32(program.Type) != 0x65a3dbe9 {
						continue
					}
					if program.Off+program.Filesz > uint64(len(result.Binary)) {
						t.Fatal("invalid syscall site table bounds")
					}
					table := result.Binary[program.Off : program.Off+program.Filesz]
					for at := 0; at+8 <= len(table); at += 8 {
						if binary.LittleEndian.Uint32(table[at+4:at+8]) == 217 {
							found = true
						}
					}
				}
				if !found {
					t.Fatal("raw syscall number missing from instruction site metadata")
				}
			}
			data, err := os.ReadFile(source)
			if err != nil {
				t.Fatal(err)
			}
			data = bytes.Replace(data, []byte("package main"), []byte("package main\nvar dynamicNumber = 217"), 1)
			data = bytes.Replace(data, []byte("syscall(217,"), []byte("syscall(dynamicNumber,"), 1)
			negative := filepath.Join(t.TempDir(), "dynamic.go")
			if err := os.WriteFile(negative, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if rejected := compile(negative); rejected.Ok {
				t.Fatal("constant-number protocol accepted a dynamic selector")
			}
		})
	}
}
