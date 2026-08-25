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

func TestCompiledInBootstrapPreparesESP32C6Definition(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s",
			runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "esp32c6.rtg")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "esp32c6/riscv32", "-tags", "m5nanoc6",
		"-o", "oracle.elf",
		filepath.Join(root, "examples", "m5nanoc6", "oracle"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
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
	if instruction := binary.LittleEndian.Uint32(image[codeOffset : codeOffset+4]); instruction&0x7f != 0x6f {
		t.Fatalf("vector-table entry instruction = %#x, want JAL", instruction)
	}
	if instruction := binary.LittleEndian.Uint32(image[codeOffset+4 : codeOffset+8]); instruction&0x7f != 0x6f {
		t.Fatalf("RMT vector instruction = %#x, want JAL", instruction)
	}
	for name, instruction := range map[string]uint32{
		"enable MIE vector one": 0x30416073,
		"retain machine mode":   0x30317073,
		"interrupt return":      0x30200073,
	} {
		encoded := make([]byte, 4)
		binary.LittleEndian.PutUint32(encoded, instruction)
		if !bytes.Contains(image, encoded) {
			t.Fatalf("ESP32-C6 image omitted %s instruction %#x", name, instruction)
		}
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
	if !containsRiscvAddressPair(image, codeOffset, len(image), bssAddress, bssEnd) {
		t.Fatalf("startup omitted clear-BSS range [%#x, %#x)", bssAddress, bssEnd)
	}
	if !bytes.Contains(image, []byte(".flash.appdesc\x00")) {
		t.Fatal("ELF omitted the ESP application descriptor section")
	}
}

func containsRiscvAddressPair(image []byte, start, end int, first, second uint32) bool {
	if end > len(image)-16 {
		end = len(image) - 16
	}
	for offset := start; offset <= end; offset += 4 {
		if riscvAddressPair(image, offset) == first &&
			riscvAddressPair(image, offset+8) == second {
			return true
		}
	}
	return false
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
	definition := filepath.Join(root, "backends", "esp32c6.rtg")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "esp32c6/riscv32",
		"-s",
		"-o", "microcontroller-suite.elf",
		filepath.Join(root, "frontend_tests", "single_file_microcontroller"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("ESP32-C6 microcontroller suite compile failed: %#v", result.Diagnostic)
	}
	if len(result.Binary) < 52 || !bytes.Equal(result.Binary[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("microcontroller suite output is not ELF32: % x",
			result.Binary[:minInt(4, len(result.Binary))])
	}
}

func TestCompiledInBootstrapCompilesESP32C6JTAGImageIntoSRAM(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "esp32c6_jtag.rtg")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "esp32c6-jtag/riscv32",
		"-tags", "m5nanoc6",
		"-s",
		"-o", "hotreload-jtag.elf",
		filepath.Join(root, "examples", "m5nanoc6", "hotreload"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("ESP32-C6 JTAG backend compile failed: %#v", result.Diagnostic)
	}
	image := result.Binary
	if len(image) < 116 || !bytes.Equal(image[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("output is not ELF32: % x", image[:minInt(4, len(image))])
	}
	if entry := binary.LittleEndian.Uint32(image[24:28]); entry != 0x40824100 {
		t.Fatalf("ELF entry = %#x, want SRAM entry 0x40824100", entry)
	}
	if text := binary.LittleEndian.Uint32(image[52+8 : 52+12]); text != 0x40824000 {
		t.Fatalf("text address = %#x, want 0x40824000", text)
	}
	dataHeader := 52 + 32
	dataEnd := binary.LittleEndian.Uint32(image[dataHeader+8:dataHeader+12]) +
		binary.LittleEndian.Uint32(image[dataHeader+20:dataHeader+24])
	if dataEnd > 0x40824000 {
		t.Fatalf("data/BSS end %#x overlaps hot-reload text", dataEnd)
	}
}

func TestCompiledInBootstrapCompilesESP32C6MixedC11Fmt(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/mixedfmt\n\ngo 1.23\n",
		"main.go": `package main

import "fmt"

func main() {
	setValue(&value)
	fmt.Printf("value: %d\n", value)
}

var value = 2
`,
		"helper.c": `extern void setValue(int *value) {
	*value = 10;
}
`,
	}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(project, name), []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	definition := filepath.Join(root, "backends", "esp32c6.rtg")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "esp32c6/riscv32",
		"-s",
		"-o", "mixed-fmt.elf",
		project,
	}, project, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("ESP32-C6 mixed C11/fmt compile failed: %#v", result.Diagnostic)
	}
	if len(result.Binary) < 52 || !bytes.Equal(result.Binary[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("mixed C11/fmt output is not ELF32: % x",
			result.Binary[:minInt(4, len(result.Binary))])
	}
}
