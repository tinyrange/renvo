package backendjit

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

func TestCompiledInBootstrapPreparesESP32P4Definition(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "esp32p4.rtg")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "esp32p4/riscv32", "-tags", "m5tab5",
		"-o", "hello.elf",
		filepath.Join(root, "examples", "m5tab5", "hello"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("ESP32-P4 custom backend compile failed: %#v", result.Diagnostic)
	}
	image := result.Binary
	if len(image) < 52 || !bytes.Equal(image[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("output is not ELF32: % x", image[:minInt(4, len(image))])
	}
	if machine := binary.LittleEndian.Uint16(image[18:20]); machine != 243 {
		t.Fatalf("ELF machine = %d, want RISC-V (243)", machine)
	}
	if entry := binary.LittleEndian.Uint32(image[24:28]); entry != 0x40000100 {
		t.Fatalf("ELF entry = %#x, want 0x40000100", entry)
	}
	if !bytes.Contains(image, []byte(".flash.appdesc\x00")) {
		t.Fatal("ELF omitted the ESP application descriptor section")
	}
}

func TestCompiledInBootstrapPreparesESP32P4CompilerHost(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "esp32p4.rtg")
	caseDir := t.TempDir()
	mainPath := filepath.Join(caseDir, "main.go")
	if err := os.WriteFile(filepath.Join(caseDir, "go.mod"),
		[]byte("module example.com/esp32p4-compiler-host\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainPath,
		[]byte("package main\nfunc main() { print(\"PASS\\n\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "esp32p4/compiler-host",
		"-o", "hello.elf",
		mainPath,
	}, caseDir, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("ESP32-P4 compiler-host compile failed: %#v", result.Diagnostic)
	}
	image := result.Binary
	if len(image) < 116 || !bytes.Equal(image[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("output is not ELF32: % x", image[:minInt(4, len(image))])
	}
	dataHeader := image[84:116]
	if address := binary.LittleEndian.Uint32(dataHeader[8:12]); address != 0x48000000 {
		t.Fatalf("compiler data address = %#x, want PSRAM 0x48000000", address)
	}
	if memory := binary.LittleEndian.Uint32(dataHeader[20:24]); memory < 28*1024*1024 || memory > 32*1024*1024 {
		t.Fatalf("compiler writable memory = %d, want a Tab5 PSRAM-backed arena", memory)
	}
	farCall := false
	codeSize := int(binary.LittleEndian.Uint32(image[68:72]))
	for offset := 0x100; offset+8 <= codeSize; offset += 4 {
		first := binary.LittleEndian.Uint32(image[offset : offset+4])
		second := binary.LittleEndian.Uint32(image[offset+4 : offset+8])
		if first&0x7f == 0x17 && second&0x7f == 0x67 {
			farCall = true
			break
		}
	}
	if !farCall {
		t.Fatal("compiler-host image omitted the full-range AUIPC/JALR call sequence")
	}
}
