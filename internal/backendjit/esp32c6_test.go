package backendjit

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

func TestCompiledInBootstrapPreparesESP32C6Definition(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("in-process prepared backend requires linux/amd64, got %s/%s",
			runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "examples", "m5nanoc6", "esp32c6.rtg")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "esp32c6/riscv32",
		"-o", "oracle.elf",
		filepath.Join(root, "examples", "m5nanoc6", "oracle"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			t.TempDir(), backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("ESP32-C6 custom backend compile failed: %#v", result.Diagnostic)
	}
	image := result.Binary
	if len(image) < 52 || !bytes.Equal(image[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("output is not ELF32: % x", image[:minInt(4, len(image))])
	}
	if machine := binary.LittleEndian.Uint16(image[18:20]); machine != 243 {
		t.Fatalf("ELF machine = %d, want RISC-V (243)", machine)
	}
	if entry := binary.LittleEndian.Uint32(image[24:28]); entry != 0x42000100 {
		t.Fatalf("ELF entry = %#x, want 0x42000100", entry)
	}
	if !bytes.Contains(image, []byte(".flash.appdesc\x00")) {
		t.Fatal("ELF omitted the ESP application descriptor section")
	}
}

func TestCompiledInBootstrapCompilesESP32C6MicrocontrollerSuite(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("in-process prepared backend requires linux/amd64, got %s/%s",
			runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "examples", "m5nanoc6", "esp32c6.rtg")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "esp32c6/riscv32",
		"-s",
		"-o", "microcontroller-suite.elf",
		filepath.Join(root, "frontend_tests", "single_file_microcontroller"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			t.TempDir(), backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("ESP32-C6 microcontroller suite compile failed: %#v", result.Diagnostic)
	}
	if len(result.Binary) < 52 || !bytes.Equal(result.Binary[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("microcontroller suite output is not ELF32: % x",
			result.Binary[:minInt(4, len(result.Binary))])
	}
}
