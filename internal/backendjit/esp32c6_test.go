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
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s",
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
	const codeOffset = 0x100
	if len(image) < codeOffset+16 {
		t.Fatalf("ELF code is too short: %d bytes", len(image))
	}
	dataHeader := 52 + 32
	bssAddress := binary.LittleEndian.Uint32(image[dataHeader+8:dataHeader+12]) +
		binary.LittleEndian.Uint32(image[dataHeader+16:dataHeader+20])
	bssAddress = (bssAddress + 7) &^ 7
	bssEnd := binary.LittleEndian.Uint32(image[dataHeader+8:dataHeader+12]) +
		binary.LittleEndian.Uint32(image[dataHeader+20:dataHeader+24])
	if bssEnd <= bssAddress || bssEnd&3 != 0 {
		t.Fatalf("final BSS range = [%#x, %#x), want non-empty 4-byte-aligned range", bssAddress, bssEnd)
	}
	if got := riscvAddressPair(image, codeOffset); got != bssAddress {
		t.Fatalf("clear-BSS start address = %#x, want %#x", got, bssAddress)
	}
	if got := riscvAddressPair(image, codeOffset+8); got != bssEnd {
		t.Fatalf("clear-BSS final address = %#x, want %#x", got, bssEnd)
	}
	if !bytes.Contains(image, []byte(".flash.appdesc\x00")) {
		t.Fatal("ELF omitted the ESP application descriptor section")
	}
}

func riscvAddressPair(image []byte, offset int) uint32 {
	upper := int32(binary.LittleEndian.Uint32(image[offset:offset+4]) & 0xfffff000)
	lower := int32(binary.LittleEndian.Uint32(image[offset+4:offset+8])) >> 20
	return uint32(upper + lower)
}

func TestCompiledInBootstrapCompilesESP32C6MicrocontrollerSuite(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s",
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
