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

func TestCompiledInBootstrapPreparesESP32S3Definition(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "esp32s3.rtg")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "esp32s3/xtensa_lx7", "-tags", "m5papermonolite",
		"-o", "oracle.elf",
		filepath.Join(root, "examples", "m5papermonolite", "oracle"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("ESP32-S3 custom backend compile failed: %#v", result.Diagnostic)
	}
	image := result.Binary
	if len(image) < 52 || !bytes.Equal(image[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("output is not ELF32: % x", image[:minInt(4, len(image))])
	}
	if machine := binary.LittleEndian.Uint16(image[18:20]); machine != 94 {
		t.Fatalf("ELF machine = %d, want Xtensa (94)", machine)
	}
	if !bytes.Contains(image, []byte("RENVO PAPERMONO-LITE PASS\n")) {
		t.Fatal("ELF omitted the serial startup oracle")
	}
	if !bytes.Contains(image, []byte(".flash.appdesc\x00")) {
		t.Fatal("ELF omitted the ESP application descriptor section")
	}
}

func TestCompiledInBootstrapCompilesPaperMonoPowerOracle(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "esp32s3.rtg")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "esp32s3/xtensa_lx7", "-tags", "m5papermonolite",
		"-s", "-o", "power-oracle.elf",
		filepath.Join(root, "examples", "m5papermonolite", "power_oracle"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("PaperMono-Lite power oracle compile failed: %#v", result.Diagnostic)
	}
	if len(result.Binary) < 52 || !bytes.Equal(result.Binary[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("power oracle output is not ELF32: % x", result.Binary[:minInt(4, len(result.Binary))])
	}
	if !bytes.Contains(result.Binary, []byte("RENVO PAPERMONO-LITE PHASE2 POWER PASS\n")) {
		t.Fatal("ELF omitted the Phase 2 power oracle")
	}
}

func TestCompiledInBootstrapCompilesPaperMonoDisplayOracle(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "esp32s3.rtg")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "esp32s3/xtensa_lx7", "-tags", "m5papermonolite",
		"-s", "-o", "display-oracle.elf",
		filepath.Join(root, "examples", "m5papermonolite", "display_oracle"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("PaperMono-Lite display oracle compile failed: %#v", result.Diagnostic)
	}
	if len(result.Binary) < 52 || !bytes.Equal(result.Binary[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("display oracle output is not ELF32: % x", result.Binary[:minInt(4, len(result.Binary))])
	}
	if !bytes.Contains(result.Binary, []byte("RENVO PAPERMONO-LITE PHASE3 FOUR GRAY PASS\n")) {
		t.Fatal("ELF omitted the Phase 3 display oracle")
	}
}

func TestCompiledInBootstrapCompilesPaperMonoTouchOracle(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "esp32s3.rtg")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "esp32s3/xtensa_lx7", "-tags", "m5papermonolite",
		"-s", "-o", "touch-oracle.elf",
		filepath.Join(root, "examples", "m5papermonolite", "touch_oracle"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("PaperMono-Lite touch oracle compile failed: %#v", result.Diagnostic)
	}
	if len(result.Binary) < 52 || !bytes.Equal(result.Binary[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("touch oracle output is not ELF32: % x", result.Binary[:minInt(4, len(result.Binary))])
	}
	if !bytes.Contains(result.Binary, []byte("RENVO PAPERMONO-LITE TOUCH READY")) {
		t.Fatal("ELF omitted the touch oracle")
	}
}

func TestCompiledInBootstrapCompilesPaperMonoTouchTrails(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "esp32s3.rtg")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "esp32s3/xtensa_lx7", "-tags", "m5papermonolite",
		"-s", "-o", "touch-trails.elf",
		filepath.Join(root, "examples", "m5papermonolite", "touch_trails"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("PaperMono-Lite touch trails compile failed: %#v", result.Diagnostic)
	}
	if len(result.Binary) < 52 || !bytes.Equal(result.Binary[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("touch trails output is not ELF32: % x", result.Binary[:minInt(4, len(result.Binary))])
	}
	if !bytes.Contains(result.Binary, []byte("RENVO PAPERMONO-LITE TOUCH TRAILS READY")) {
		t.Fatal("ELF omitted the touch trails example")
	}
}

func TestCompiledInBootstrapCompilesPaperMonoFormsDemo(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "esp32s3.rtg")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "esp32s3/xtensa_lx7", "-tags", "m5papermonolite",
		"-arena-size", "90112", "-s", "-o", "forms-demo.elf",
		filepath.Join(root, "examples", "m5papermonolite", "forms_demo"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("PaperMono-Lite Forms demo compile failed: %#v", result.Diagnostic)
	}
	if len(result.Binary) < 52 || !bytes.Equal(result.Binary[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("Forms demo output is not ELF32: % x", result.Binary[:minInt(4, len(result.Binary))])
	}
	if !bytes.Contains(result.Binary, []byte("RENVO PAPERMONO-LITE FORMS READY")) {
		t.Fatal("ELF omitted the Forms demo startup marker")
	}
}
