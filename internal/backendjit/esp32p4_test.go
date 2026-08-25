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
